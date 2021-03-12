package jenkins

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccJenkinsCredentialStringDataSource_basic(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource jenkins_credential_string foo {
				  name = "tf-acc-test-%s"
				  description = "Terraform acceptance tests %s"
				  secret = "bar"
				}

				data jenkins_credential_string foo {
					name   = jenkins_credential_string.foo.name
					domain = "_"
				}`, randString, randString),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jenkins_credential_string.foo", "id", "/tf-acc-test-"+randString),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.foo", "id", "/tf-acc-test-"+randString),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.foo", "name", "tf-acc-test-"+randString),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.foo", "description", "Terraform acceptance tests "+randString),
				),
			},
		},
	})
}

func TestAccJenkinsCredentialStringDataSource_nested(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource jenkins_folder foo {
					name = "tf-acc-test-%s"
				}

				resource jenkins_credential_string sub {
					name = "subfolder"
					folder = jenkins_folder.foo.id
					description = "Terraform acceptance tests %s"
					secret = "bar"
				}

				data jenkins_credential_string sub {
					name   = jenkins_credential_string.sub.name
					domain = "_"
					folder = jenkins_credential_string.sub.folder
				}`, randString, randString),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jenkins_folder.foo", "id", "/job/tf-acc-test-"+randString),
					resource.TestCheckResourceAttr("jenkins_credential_string.sub", "id", "/job/tf-acc-test-"+randString+"/subfolder"),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.sub", "name", "subfolder"),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.sub", "folder", "/job/tf-acc-test-"+randString),
					resource.TestCheckResourceAttr("data.jenkins_credential_string.sub", "description", "Terraform acceptance tests "+randString),
				),
			},
		},
	})
}
