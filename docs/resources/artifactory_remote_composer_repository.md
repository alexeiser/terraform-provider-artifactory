# Artifactory Remote PHP Composer Repository Resource

Creates a remote PHP Composer repository.
Official documentation can be found [here](https://www.jfrog.com/confluence/display/JFROG/PHP+Composer+Repositories)


## Example Usage
To create a new Artifactory remote PHP Composer repository called my-remote-composer.

```hcl
resource "artifactory_remote_composer_repository" "my-remote-composer" {
  key                         = "my-remote-composer"
  url                         = "https://github.com/"
  vcs_git_provider            = "GITHUB"
}
```

## Argument Reference

Arguments have a one to one mapping with the [JFrog API](https://www.jfrog.com/confluence/display/RTF/Repository+Configuration+JSON). The following arguments are supported:

* `key` - (Required) The repository identifier. Must be unique system-wide
* `description` - (Optional)
* `notes` - (Optional)
* `url` - (Required) - the remote repo URL. You kinda don't have a remote repo without it
* `vcs_git_provider` - (Optional) Artifactory supports proxying the following Git providers out-of-the-box: GitHub or a remote Artifactory instance. Default value is "ARTIFACTORY".
* `vcs_git_download_url` - (Optional) This attribute is used when vcs_git_provider is set to 'CUSTOM'. Provided URL will be used as proxy.
* `composer_registry_url` - (Optional) Proxy remote Composer repository. Default value is "https://packagist.org".

Arguments for remote PHP Composer repository type closely match with arguments for remote Generic repository type.