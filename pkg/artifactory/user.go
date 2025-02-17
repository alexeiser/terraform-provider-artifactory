package artifactory

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type User struct {
	Name                     string   `json:"name"`
	Email                    string   `json:"email"`
	Password                 string `json:"password,omitempty"`
	Admin                    bool     `json:"admin"`
	ProfileUpdatable         bool     `json:"profileUpdatable"`
	DisableUIAccess          bool     `json:"disableUIAccess"`
	InternalPasswordDisabled bool     `json:"internalPasswordDisabled"`
	LastLoggedIn             string   `json:"lastLoggedIn"`
	Realm                    string   `json:"realm"`
	Groups                   []string `json:"groups"`
}

var baseUserSchema = map[string]*schema.Schema{
	"name": {
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
		Description:      "(Required) Username for user.",
	},
	"email": {
		Type:             schema.TypeString,
		Required:         true,
		ValidateDiagFunc: validation.ToDiagFunc(validateIsEmail),
		Description:      "(Required) Email for user.",
	},
	"admin": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "(Optional, Default: false) When enabled, this user is an administrator with all the ensuing privileges.",
	},
	"profile_updatable": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
		Description: "(Optional, Default: false) When enabled, this user can update their profile details (except for the password. " +
			"Only an administrator can update the password). There may be cases in which you want to leave " +
			"this unset to prevent users from updating their profile. For example, a departmental user with " +
			"a single password shared between all department members.",
	},
	"disable_ui_access": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
		Description: "(Optional, Default: false) When enabled, this user can only access the system through the REST API." +
			" This option cannot be set if the user has Admin privileges.",
	},
	"internal_password_disabled": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
		Description: "(Optional, Default: false) When enabled, disables the fallback mechanism for using an internal password when " +
			"external authentication (such as LDAP) is enabled.",
	},
	"groups": {
		Type:        schema.TypeSet,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Set:         schema.HashString,
		Optional:    true,
		Description: "(Optional) List of groups this user is a part of.",
	},
}

func unpackUser(s *schema.ResourceData) User {
	d := &ResourceData{s}
	return User{
		Name:                     d.getString("name", false),
		Email:                    d.getString("email", false),
		Password:                 d.getString("password", false),
		Admin:                    d.getBool("admin", false),
		ProfileUpdatable:         d.getBool("profile_updatable", false),
		DisableUIAccess:          d.getBool("disable_ui_access", false),
		InternalPasswordDisabled: d.getBool("internal_password_disabled", false),
		Groups:                   d.getSet("groups"),
	}
}

func packUser(user User, d *schema.ResourceData) diag.Diagnostics {

	setValue := mkLens(d)

	setValue("name", user.Name)
	setValue("email", user.Email)
	setValue("admin", user.Admin)
	setValue("profile_updatable", user.ProfileUpdatable)
	setValue("disable_ui_access", user.DisableUIAccess)
	errors := setValue("internal_password_disabled", user.InternalPasswordDisabled)

	if user.Groups != nil {
		errors = setValue("groups", schema.NewSet(schema.HashString, castToInterfaceArr(user.Groups)))
	}

	if errors != nil && len(errors) > 0 {
		return diag.Errorf("failed to pack user %q", errors)
	}

	return nil
}

const usersEndpointPath = "artifactory/api/security/users/"

func resourceUserRead(ctx context.Context, rd *schema.ResourceData, m interface{}) diag.Diagnostics {
	d := &ResourceData{rd}

	userName := d.Id()
	user := &User{}
	resp, err := m.(*resty.Client).R().SetResult(user).Get(usersEndpointPath + userName)

	if err != nil {
		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return packUser(*user, rd)
}

func resourceBaseUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}, passwordGenerator func(*User) diag.Diagnostics) diag.Diagnostics {
	user := unpackUser(d)

	var diags diag.Diagnostics

	if passwordGenerator != nil {
		diags = passwordGenerator(&user)
	}

	_, err := m.(*resty.Client).R().SetBody(user).Put(usersEndpointPath + user.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	// Artifactory PUT call for creating user with groups attribute set to empty/null always sets groups to "readers".
	// This is a bug on Artifactory. Below workaround will fix the issue and has to be removed after the artifactory bug is resolved.
	// Workaround: We use following POST call to update the user's groups config to empty group.
	// This action will match the expectation for this resource when "groups" attribute is empty or not specified in hcl.
	if user.Groups == nil {
		user.Groups = []string{}
		_, errGroupUpdate := m.(*resty.Client).R().SetBody(user).Post(usersEndpointPath + user.Name)
		if errGroupUpdate != nil {
			return diag.FromErr(errGroupUpdate)
		}
	}

	d.SetId(user.Name)

	retryError := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		result := &User{}
		resp, e := m.(*resty.Client).R().SetResult(result).Get(usersEndpointPath + user.Name)

		if e != nil {
			if resp != nil && resp.StatusCode() == http.StatusNotFound {
				return resource.RetryableError(fmt.Errorf("expected user to be created, but currently not found"))
			}
			return resource.NonRetryableError(fmt.Errorf("error describing user: %s", err))
		}

		packUser(*result, d)

		return nil
	})

	if retryError != nil {
		return diag.FromErr(retryError)
	}

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	user := unpackUser(d)
	_, err := m.(*resty.Client).R().SetBody(user).Post(usersEndpointPath + user.Name)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(user.Name)
	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, rd *schema.ResourceData, m interface{}) diag.Diagnostics {
	d := &ResourceData{rd}
	userName := d.getString("name", false)

	_, err := m.(*resty.Client).R().Delete(usersEndpointPath + userName)
	if err != nil {
		return diag.Errorf("user %s not deleted. %s", userName, err)
	}

	d.SetId("")

	return nil
}

func resourceUserExists(data *schema.ResourceData, m interface{}) (bool, error) {
	d := &ResourceData{data}
	name := d.Id()

	resp, err := m.(*resty.Client).R().Head(usersEndpointPath + name)
	if err != nil && resp != nil && resp.StatusCode() == http.StatusNotFound {
		// Do not error on 404s as this causes errors when the upstream user has been manually removed
		return false, nil
	}

	return err == nil, err
}
