package jenkins

import (
	"fmt"
	"testing"

	jenkins "github.com/bndr/gojenkins"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccJenkinsCredentialString_basic(t *testing.T) {
	var cred jenkins.StringCredentials

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckJenkinsCredentialStringDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
				resource jenkins_credential_string foo {
				  name = "test-username"
				  secret = "bar"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jenkins_credential_string.foo", "id", "/test-username"),
					testAccCheckJenkinsCredentialStringExists("jenkins_credential_string.foo", &cred),
				),
			},
			{
				// Update by adding description
				Config: `
				resource jenkins_credential_string foo {
				  name = "test-username"
				  description = "new-description"
				  secret = "bar"
				}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckJenkinsCredentialStringExists("jenkins_credential_string.foo", &cred),
					resource.TestCheckResourceAttr("jenkins_credential_string.foo", "description", "new-description"),
				),
			},
		},
	})
}

func TestAccJenkinsCredentialString_folder(t *testing.T) {
	var cred jenkins.StringCredentials
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckJenkinsCredentialStringDestroy,
			testAccCheckJenkinsFolderDestroy,
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource jenkins_folder foo {
					name = "tf-acc-test-%s"
					description = "Terraform acceptance testing"
				}

				resource jenkins_folder foo_sub {
					name = "subfolder"
					folder = jenkins_folder.foo.id
					description = "Terraform acceptance testing"
				}

				resource jenkins_credential_string foo {
				  name = "test-username"
				  folder = jenkins_folder.foo_sub.id
				  secret = "bar"
				}`, randString),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jenkins_credential_string.foo", "id", "/job/tf-acc-test-"+randString+"/job/subfolder/test-username"),
					testAccCheckJenkinsCredentialStringExists("jenkins_credential_string.foo", &cred),
				),
			},
			{
				// Update by adding description
				Config: fmt.Sprintf(`
				resource jenkins_folder foo {
					name = "tf-acc-test-%s"
					description = "Terraform acceptance testing"

					lifecycle {
						ignore_changes = [template]
					}
				}

				resource jenkins_folder foo_sub {
					name = "subfolder"
					folder = jenkins_folder.foo.id
					description = "Terraform acceptance testing"

					lifecycle {
						ignore_changes = [template]
					}
				}

				resource jenkins_credential_string foo {
				  name = "test-username"
				  folder = jenkins_folder.foo_sub.id
				  description = "new-description"
				  secret = "bar"
				}`, randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckJenkinsCredentialStringExists("jenkins_credential_string.foo", &cred),
					resource.TestCheckResourceAttr("jenkins_credential_string.foo", "description", "new-description"),
				),
			},
		},
	})
}

func testAccCheckJenkinsCredentialStringExists(resourceName string, cred *jenkins.StringCredentials) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(jenkinsClient)

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf(resourceName + " not found")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID is not set")
		}

		manager := client.Credentials()
		manager.Folder = formatFolderName(rs.Primary.Attributes["folder"])
		err := manager.GetSingle(rs.Primary.Attributes["domain"], rs.Primary.Attributes["name"], cred)
		if err != nil {
			return fmt.Errorf("Unable to retrieve credentials for %s - %s: %w", rs.Primary.Attributes["folder"], rs.Primary.Attributes["name"], err)
		}

		return nil
	}
}

func testAccCheckJenkinsCredentialStringDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(jenkinsClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "jenkins_credential_string" {
			continue
		} else if _, ok := rs.Primary.Meta["name"]; !ok {
			continue
		}

		cred := jenkins.StringCredentials{}
		manager := client.Credentials()
		manager.Folder = formatFolderName(rs.Primary.Meta["folder"].(string))
		err := manager.GetSingle(rs.Primary.Meta["domain"].(string), rs.Primary.Meta["name"].(string), &cred)
		if err == nil {
			return fmt.Errorf("Credentials still exists: %s - %s", rs.Primary.Attributes["folder"], rs.Primary.Attributes["name"])
		}
	}

	return nil
}
