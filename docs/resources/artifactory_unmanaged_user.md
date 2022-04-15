# Artifactory Unmanaged User Resource

Provides an Artifactory unmanaged user resource. This can be used to create and maintain Artifactory users.

When the optional attribute `password` is omitted, a random password is generated according to current Artifactory password policy.

## Example Usage

```hcl
# Create a new Artifactory user called terraform
resource "artifactory_unmanaged_user" "test-user" {
  name     = "terraform"
  email    = "test-user@artifactory-terraform.com"
  groups   = ["logged-in-users", "readers"]
  password = "my super secret password"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Username for user.
* `email` - (Required) Email for user.
* `password` - (Optional) Password for the user. When omitted, a random password is generated using the following password policy: 10 characters with 1 digit, 1 symbol, with upper and lower case letters.
* `admin` - (Optional) When enabled, this user is an administrator with all the ensuing privileges. Default value is `false`.
* `profile_updatable` - (Optional) When set, this user can update his profile details (except for the password. Only an administrator can update the password). Default value is `true`.
* `disable_ui_access` - (Optional) When set, this user can only access Artifactory through the REST API. This option cannot be set if the user has Admin privileges. Default value is `true`.
* `internal_password_disabled` - (Optional) When set, disables the fallback of using an internal password when external authentication (such as LDAP) is enabled.
* `groups` - (Optional) List of groups this user is a part of.

## Import

Users can be imported using their name, e.g.

```
$ terraform import artifactory_unmanaged_user.test-user myusername
```
