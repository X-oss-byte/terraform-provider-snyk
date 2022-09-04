package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/pavel-snyk/snyk-sdk-go/snyk"
)

func TestAccResourceIntegration_basic(t *testing.T) {
	t.Parallel()

	var integration snyk.Integration
	organizationName := fmt.Sprintf("tf-test-acc_%s", acctest.RandString(10))
	groupID := os.Getenv("SNYK_GROUP_ID")
	token := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceIntegrationConfig(organizationName, groupID, ""),
				ExpectError: regexp.MustCompile("Wrong credentials for given integration type"),
			},
			{
				Config: testAccResourceIntegrationConfig(organizationName, groupID, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckResourceIntegrationExists("snyk_integration.test", organizationName, &integration),
					resource.TestCheckResourceAttrSet("snyk_integration.test", "id"),
					resource.TestCheckResourceAttr("snyk_integration.test", "type", "gitlab"),
				),
			},
		},
	})
}

func testAccCheckResourceIntegrationExists(resourceName, organizationName string, integration *snyk.Integration) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		// retrieve resource from state
		rs := state.RootModule().Resources[resourceName]

		if rs.Primary.ID == "" {
			return fmt.Errorf("integration ID is not set")
		}

		client := testAccProvider.(*snykProvider).client
		orgs, _, err := client.Orgs.List(context.Background())
		if err != nil {
			return err
		}

		organizationID := ""
		for _, org := range orgs {
			if org.Name == organizationName {
				organizationID = org.ID
				return nil
			}
		}
		if organizationID == "" {
			return fmt.Errorf("organization (%s) for integration (%s) not found", organizationName, rs.Primary.ID)
		}

		integrations, _, err := client.Integrations.List(context.Background(), organizationID)
		for t, id := range integrations {
			if id == rs.Primary.ID {
				integration = &snyk.Integration{
					ID:   id,
					Type: t,
				}
				return nil
			}
		}

		return fmt.Errorf("integration (%s) not found", rs.Primary.ID)
	}
}

func testAccResourceIntegrationConfig(organizationName, groupID, token string) string {
	return fmt.Sprintf(`
resource "snyk_organization" "test" {
  name = "%s"
  group_id = "%s"
}
resource "snyk_integration" "test" {
  organization_id = snyk_organization.test.id

  type  = "gitlab"
  url   = "https://testing.gitlab.local"
  token = "%s"
}
`, organizationName, groupID, token)
}
