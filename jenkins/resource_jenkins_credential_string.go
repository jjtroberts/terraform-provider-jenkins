package jenkins

import (
	"context"
	"fmt"
	"strings"

	jenkins "github.com/bndr/gojenkins"
	//"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//var supportedCredentialScopes = []string{"SYSTEM", "GLOBAL"}

func resourceJenkinsCredentialString() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceJenkinsCredentialStringCreate,
		ReadContext:   resourceJenkinsCredentialStringRead,
		UpdateContext: resourceJenkinsCredentialStringUpdate,
		DeleteContext: resourceJenkinsCredentialStringDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceJenkinsCredentialStringImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The identifier assigned to the credentials.",
				Required:    true,
				ForceNew:    true,
			},
			"domain": {
				Type:        schema.TypeString,
				Description: "The domain namespace that the credentials will be added to.",
				Optional:    true,
				Default:     "_",
				// In-place updates should be possible, but gojenkins does not support move operations
				ForceNew: true,
			},
			"folder": {
				Type:        schema.TypeString,
				Description: "The folder namespace that the credentials will be added to.",
				Optional:    true,
				ForceNew:    true,
			},
			"scope": {
				Type:             schema.TypeString,
				Description:      "The Jenkins scope assigned to the credentials.",
				Optional:         true,
				Default:          "GLOBAL",
				ValidateDiagFunc: validateCredentialScope,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "The credentials descriptive text.",
				Optional:    true,
				Default:     "Managed by Terraform",
			},
			"secret": {
				Type:        schema.TypeString,
				Description: "The credentials secret text string. If left empty will be unmanaged.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceJenkinsCredentialStringCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(jenkinsClient)
	cm := client.Credentials()
	cm.Folder = formatFolderName(d.Get("folder").(string))

	// Validate that the folder exists
	if err := folderExists(client, cm.Folder); err != nil {
		return diag.FromErr(fmt.Errorf("invalid folder name '%s' specified: %w", cm.Folder, err))
	}

	cred := jenkins.StringCredentials{
		ID:          d.Get("name").(string),
		Scope:       d.Get("scope").(string),
		Description: d.Get("description").(string),
		Secret:      d.Get("secret").(string),
	}

	domain := d.Get("domain").(string)
	err := cm.Add(domain, cred)
	if err != nil {
		return diag.Errorf("Could not create String credentials: %s", err)
	}

	d.SetId(generateCredentialID(d.Get("folder").(string), cred.ID))
	return resourceJenkinsCredentialStringRead(ctx, d, meta)
}

func resourceJenkinsCredentialStringRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cm := meta.(jenkinsClient).Credentials()
	cm.Folder = formatFolderName(d.Get("folder").(string))

	cred := jenkins.StringCredentials{}
	err := cm.GetSingle(
		d.Get("domain").(string),
		d.Get("name").(string),
		&cred,
	)

	if err != nil {
		if strings.HasSuffix(err.Error(), "404") {
			// Job does not exist
			d.SetId("")
			return nil
		}

		return diag.Errorf("Could not read String credentials: %s", err)
	}

	d.SetId(generateCredentialID(d.Get("folder").(string), cred.ID))
	d.Set("scope", cred.Scope)
	d.Set("description", cred.Description)
	// NOTE: We are NOT setting the password here, as the password returned by GetSingle is garbage
	// Password only applies to Create/Update operations if the "password" property is non-empty

	return nil
}

func resourceJenkinsCredentialStringUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cm := meta.(jenkinsClient).Credentials()
	cm.Folder = formatFolderName(d.Get("folder").(string))

	domain := d.Get("domain").(string)
	cred := jenkins.StringCredentials{
		ID:          d.Get("name").(string),
		Scope:       d.Get("scope").(string),
		Description: d.Get("description").(string),
	}

	// Only enforce the string if it is non-empty
	if d.Get("secret").(string) != "" {
		cred.Secret = d.Get("secret").(string)
	}

	err := cm.Update(domain, d.Get("name").(string), &cred)
	if err != nil {
		return diag.Errorf("Could not update String credentials: %s", err)
	}

	d.SetId(generateCredentialID(d.Get("folder").(string), cred.ID))
	return resourceJenkinsCredentialStringRead(ctx, d, meta)
}

func resourceJenkinsCredentialStringDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cm := meta.(jenkinsClient).Credentials()
	cm.Folder = formatFolderName(d.Get("folder").(string))

	err := cm.Delete(
		d.Get("domain").(string),
		d.Get("name").(string),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceJenkinsCredentialStringImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	ret := []*schema.ResourceData{d}

	splitID := strings.Split(d.Id(), "/")
	if len(splitID) < 2 {
		return ret, fmt.Errorf("import ID was improperly formatted. Imports need to be in the format \"[<folder>/]<domain>/<name>\"")
	}

	name := splitID[len(splitID)-1]
	d.Set("name", name)

	domain := splitID[len(splitID)-2]
	d.Set("domain", domain)

	folder := strings.Trim(strings.Join(splitID[0:len(splitID)-2], "/"), "/")
	d.Set("folder", folder)

	d.SetId(generateCredentialID(folder, name))
	return ret, nil
}

// func validateCredentialScope(v interface{}, p cty.Path) diag.Diagnostics {
// 	for _, supported := range supportedCredentialScopes {
// 		if v == supported {
// 			return nil
// 		}
// 	}
// 	return diag.Errorf("Invalid scope: %s. Supported scopes are: %s", v, strings.Join(supportedCredentialScopes, ", "))
// }

// func generateCredentialID(folder, name string) string {
// 	return fmt.Sprintf("%s/%s", folder, name)
// }
