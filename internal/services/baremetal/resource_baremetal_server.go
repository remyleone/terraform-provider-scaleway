package baremetal

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	sdkValidation "github.com/scaleway/scaleway-sdk-go/validation"
)

func ResourceScalewayBaremetalServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayBaremetalServerCreate,
		ReadContext:   ResourceScalewayBaremetalServerRead,
		UpdateContext: ResourceScalewayBaremetalServerUpdate,
		DeleteContext: ResourceScalewayBaremetalServerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultBaremetalServerTimeout),
			Create:  schema.DefaultTimeout(defaultBaremetalServerTimeout),
			Update:  schema.DefaultTimeout(defaultBaremetalServerTimeout),
			Delete:  schema.DefaultTimeout(defaultBaremetalServerTimeout),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Name of the server",
			},
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Hostname of the server",
			},
			"offer": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID or name of the server offer",
				DiffSuppressFunc: func(_, oldValue, newValue string, d *schema.ResourceData) bool {
					// remove the locality from the IDs when checking diff
					if locality.ExpandID(newValue) == locality.ExpandID(oldValue) {
						return true
					}
					// if the offer was provided by name
					offerName, ok := d.GetOk("offer_name")
					return ok && newValue == offerName
				},
			},
			"offer_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the server offer",
			},
			"offer_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the server offer",
			},
			"os": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The base image of the server",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
			},
			"os_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The base image name of the server",
			},
			"ssh_key_ids": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: verify.UUID(),
				},
				Optional: true,
				Description: `Array of SSH key IDs allowed to SSH to the server

**NOTE** : If you are attempting to update your SSH key IDs, it will induce the reinstall of your server. 
If this behaviour is wanted, please set 'reinstall_on_ssh_key_changes' argument to true.`,
			},
			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User used for the installation.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password used for the installation.",
			},
			"service_user": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User used for the service to install.",
			},
			"service_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password used for the service to install.",
			},
			"reinstall_on_config_changes": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If True, this boolean allows to reinstall the server on SSH key IDs, user or password changes",
			},
			"install_config_afterward": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If True, this boolean allows to create a server without the install config if you want to provide it later",
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 255),
				Description:  "Some description to associate to the server, max 255 characters",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Computed:    true,
				Description: "Array of tags to associate with the server",
			},
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
			"ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IP addresses attached to the server.",
				Elem:        ResourceScalewayBaremetalServerIP(),
			},
			"ipv4": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPv4 addresses attached to the server",
				Elem:        ResourceScalewayBaremetalServerIP(),
			},
			"ipv6": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPv6 addresses attached to the server",
				Elem:        ResourceScalewayBaremetalServerIP(),
			},
			"domain": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"options": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The options to enable on server",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "IDs of the options",
							Required:    true,
						},
						"expires_at": {
							Type:             schema.TypeString,
							Description:      "Auto expire the option after this date",
							Optional:         true,
							Computed:         true,
							ValidateDiagFunc: verify.ValidateDate(),
							DiffSuppressFunc: difffuncs.DiffSuppressFuncTimeRFC3339,
						},
						// computed
						"name": {
							Type:        schema.TypeString,
							Description: "name of the option",
							Computed:    true,
						},
					},
				},
			},
			"private_network": {
				Type:        schema.TypeSet,
				Optional:    true,
				Set:         baremetalPrivateNetworkSetHash,
				Description: "The private networks to attach to the server",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Description:  "The private network ID",
							Required:     true,
							ValidateFunc: locality.UUIDorUUIDWithLocality(),
							StateFunc: func(i interface{}) string {
								return locality.ExpandID(i.(string))
							},
						},
						// computed
						"vlan": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The VLAN ID associated to the private network",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The private network status",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the creation of the private network",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the last update of the private network",
						},
					},
				},
			},
		},
		CustomizeDiff: customdiff.Sequence(
			difffuncs.CustomizeDiffLocalityCheck("private_network.#.id"),
			customDiffBaremetalPrivateNetworkOption(),
		),
	}
}

func ResourceScalewayBaremetalServerIP() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the IPv6",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the IPv6",
			},
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IPv6 address",
			},
			"reverse": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Reverse of the IPv6",
			},
		},
	}
}

func ResourceScalewayBaremetalServerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	baremetalAPI, zone, err := BaremetalAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	baremetalPrivateNetworkAPI, _, err := BaremetalPrivateNetworkAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	offerID := locality.ExpandZonedID(d.Get("offer"))
	if !sdkValidation.IsUUID(offerID.ID) {
		o, err := baremetalAPI.GetOfferByName(&baremetal.GetOfferByNameRequest{
			OfferName: offerID.ID,
			Zone:      zone,
		})
		if err != nil {
			return diag.FromErr(err)
		}
		offerID = locality.NewZonedID(zone, o.ID)
	}

	if !d.Get("install_config_afterward").(bool) {
		if diags := validateInstallConfig(ctx, d, meta); len(diags) > 0 {
			return diags
		}
	}

	server, err := baremetalAPI.CreateServer(&baremetal.CreateServerRequest{
		Zone:        zone,
		Name:        types.ExpandOrGenerateString(d.Get("name"), "bm"),
		ProjectID:   types.ExpandStringPtr(d.Get("project_id")),
		Description: d.Get("description").(string),
		OfferID:     offerID.ID,
		Tags:        types.ExpandStrings(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedID(server.Zone, server.ID).String())

	_, err = waitForBaremetalServer(ctx, baremetalAPI, zone, server.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.Get("install_config_afterward").(bool) {
		_, err = baremetalAPI.InstallServer(&baremetal.InstallServerRequest{
			Zone:            server.Zone,
			ServerID:        server.ID,
			OsID:            locality.ExpandZonedID(d.Get("os")).ID,
			Hostname:        types.ExpandStringWithDefault(d.Get("hostname"), server.Name),
			SSHKeyIDs:       types.ExpandStrings(d.Get("ssh_key_ids")),
			User:            types.ExpandStringPtr(d.Get("user")),
			Password:        types.ExpandStringPtr(d.Get("password")),
			ServiceUser:     types.ExpandStringPtr(d.Get("service_user")),
			ServicePassword: types.ExpandStringPtr(d.Get("service_password")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForBaremetalServerInstall(ctx, baremetalAPI, zone, server.ID, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	options, optionsExist := d.GetOk("options")
	if optionsExist {
		opSpecs, err := expandBaremetalOptions(options)
		if err != nil {
			return diag.FromErr(err)
		}
		for i := range opSpecs {
			_, err = baremetalAPI.AddOptionServer(&baremetal.AddOptionServerRequest{
				Zone:      server.Zone,
				ServerID:  server.ID,
				OptionID:  opSpecs[i].ID,
				ExpiresAt: opSpecs[i].ExpiresAt,
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	privateNetworkIDs, pnExist := d.GetOk("private_network")
	if pnExist {
		createBaremetalPrivateNetworkRequest := &baremetal.PrivateNetworkAPISetServerPrivateNetworksRequest{
			Zone:              zone,
			ServerID:          server.ID,
			PrivateNetworkIDs: expandBaremetalPrivateNetworks(privateNetworkIDs),
		}

		baremetalPrivateNetwork, err := baremetalPrivateNetworkAPI.SetServerPrivateNetworks(
			createBaremetalPrivateNetworkRequest,
			scw.WithContext(ctx),
		)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForBaremetalServerPrivateNetwork(ctx, baremetalPrivateNetworkAPI, zone, baremetalPrivateNetwork.ServerPrivateNetworks[0].ServerID, d.Timeout(schema.TimeoutCreate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayBaremetalServerRead(ctx, d, meta)
}

func ResourceScalewayBaremetalServerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	baremetalAPI, zonedID, err := BaremetalAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	baremetalPrivateNetworkAPI, _, err := BaremetalPrivateNetworkAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := baremetalAPI.GetServer(&baremetal.GetServerRequest{
		Zone:     zonedID.Zone,
		ServerID: zonedID.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	offer, err := baremetalAPI.GetOffer(&baremetal.GetOfferRequest{
		Zone:    server.Zone,
		OfferID: server.OfferID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	var os *baremetal.OS
	if server.Install != nil {
		os, err = baremetalAPI.GetOS(&baremetal.GetOSRequest{
			Zone: server.Zone,
			OsID: server.Install.OsID,
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	_ = d.Set("name", server.Name)
	_ = d.Set("zone", server.Zone.String())
	_ = d.Set("organization_id", server.OrganizationID)
	_ = d.Set("project_id", server.ProjectID)
	_ = d.Set("offer_id", locality.NewZonedIDString(server.Zone, offer.ID))
	_ = d.Set("offer_name", offer.Name)
	_ = d.Set("offer", locality.NewZonedIDString(server.Zone, offer.ID))
	_ = d.Set("tags", server.Tags)
	_ = d.Set("domain", server.Domain)
	_ = d.Set("ips", flattenBaremetalIPs(server.IPs))
	_ = d.Set("ipv4", flattenBaremetalIPv4s(server.IPs))
	_ = d.Set("ipv6", flattenBaremetalIPv6s(server.IPs))
	if server.Install != nil {
		_ = d.Set("os", locality.NewZonedIDString(server.Zone, os.ID))
		_ = d.Set("os_name", os.Name)
		_ = d.Set("ssh_key_ids", server.Install.SSHKeyIDs)
		_ = d.Set("user", server.Install.User)
		_ = d.Set("service_user", server.Install.ServiceUser)
	}
	_ = d.Set("description", server.Description)
	_ = d.Set("options", flattenBaremetalOptions(server.Zone, server.Options))

	listPrivateNetworks, err := baremetalPrivateNetworkAPI.ListServerPrivateNetworks(&baremetal.PrivateNetworkAPIListServerPrivateNetworksRequest{
		Zone:     server.Zone,
		ServerID: &server.ID,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list server's private networks: %w", err))
	}
	pnRegion, err := server.Zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}
	_ = d.Set("private_network", flattenBaremetalPrivateNetworks(pnRegion, listPrivateNetworks.ServerPrivateNetworks))

	return nil
}

//gocyclo:ignore
func ResourceScalewayBaremetalServerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	baremetalAPI, zonedID, err := BaremetalAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	baremetalPrivateNetworkAPI, zone, err := BaremetalPrivateNetworkAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := baremetalAPI.GetServer(&baremetal.GetServerRequest{
		Zone:     zonedID.Zone,
		ServerID: zonedID.ID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	var serverGetOptionIDs []*baremetal.ServerOption
	serverGetOptionIDs = append(serverGetOptionIDs, server.Options...)

	if d.HasChange("options") {
		options, err := expandBaremetalOptions(d.Get("options"))
		if err != nil {
			return diag.FromErr(err)
		}
		optionsToDelete := baremetalCompareOptions(options, serverGetOptionIDs)
		for i := range optionsToDelete {
			_, err = baremetalAPI.DeleteOptionServer(&baremetal.DeleteOptionServerRequest{
				Zone:     server.Zone,
				ServerID: server.ID,
				OptionID: optionsToDelete[i].ID,
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}

		_, err = waitForBaremetalServerOptions(ctx, baremetalAPI, zonedID.Zone, zonedID.ID, d.Timeout(schema.TimeoutDelete))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		optionsToAdd := baremetalCompareOptions(serverGetOptionIDs, options)
		for i := range optionsToAdd {
			_, err = baremetalAPI.AddOptionServer(&baremetal.AddOptionServerRequest{
				Zone:      server.Zone,
				ServerID:  server.ID,
				OptionID:  optionsToAdd[i].ID,
				ExpiresAt: optionsToAdd[i].ExpiresAt,
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("private_network") {
		privateNetworkIDs := d.Get("private_network")

		updateBaremetalPrivateNetworkRequest := &baremetal.PrivateNetworkAPISetServerPrivateNetworksRequest{
			Zone:              zone,
			ServerID:          server.ID,
			PrivateNetworkIDs: expandBaremetalPrivateNetworks(privateNetworkIDs),
		}

		baremetalPrivateNetwork, err := baremetalPrivateNetworkAPI.SetServerPrivateNetworks(
			updateBaremetalPrivateNetworkRequest,
			scw.WithContext(ctx),
		)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForBaremetalServerPrivateNetwork(ctx, baremetalPrivateNetworkAPI, zone, baremetalPrivateNetwork.ServerPrivateNetworks[0].ServerID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	req := &baremetal.UpdateServerRequest{
		Zone:     zonedID.Zone,
		ServerID: zonedID.ID,
	}

	hasChanged := false

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
		hasChanged = true
	}

	if d.HasChange("description") {
		req.Description = types.ExpandUpdatedStringPtr(d.Get("description"))
		hasChanged = true
	}

	if d.HasChange("tags") {
		req.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
		hasChanged = true
	}

	if hasChanged {
		_, err = baremetalAPI.UpdateServer(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	installReq := &baremetal.InstallServerRequest{
		Zone:            zonedID.Zone,
		ServerID:        zonedID.ID,
		Hostname:        types.ExpandStringWithDefault(d.Get("hostname"), d.Get("name").(string)),
		SSHKeyIDs:       types.ExpandStrings(d.Get("ssh_key_ids")),
		User:            types.ExpandStringPtr(d.Get("user")),
		Password:        types.ExpandStringPtr(d.Get("password")),
		ServiceUser:     types.ExpandStringPtr(d.Get("service_user")),
		ServicePassword: types.ExpandStringPtr(d.Get("service_password")),
	}

	if d.HasChange("os") {
		if diags := validateInstallConfig(ctx, d, meta); len(diags) > 0 {
			return diags
		}
		err = baremetalInstallServer(ctx, d, baremetalAPI, installReq)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForBaremetalServerInstall(ctx, baremetalAPI, zonedID.Zone, zonedID.ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	var diags diag.Diagnostics

	if d.HasChanges("ssh_key_ids", "user", "password", "reinstall_on_config_changes") {
		if !d.Get("reinstall_on_config_changes").(bool) && !d.HasChange("os") {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Changes have been made on your config",
				Detail: "[WARN] This change induce the reinstall of your server. " +
					"If this behaviour is wanted, please set 'reinstall_on_config_changes' argument to true",
			})
		} else {
			if diags := validateInstallConfig(ctx, d, meta); len(diags) > 0 {
				return diags
			}
			err = baremetalInstallServer(ctx, d, baremetalAPI, installReq)
			if err != nil {
				return diag.FromErr(err)
			}

			_, err = waitForBaremetalServerInstall(ctx, baremetalAPI, zonedID.Zone, zonedID.ID, d.Timeout(schema.TimeoutUpdate))
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return append(diags, ResourceScalewayBaremetalServerRead(ctx, d, meta)...)
}

func ResourceScalewayBaremetalServerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	baremetalAPI, zonedID, err := BaremetalAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = detachAllPrivateNetworkFromBaremetal(ctx, d, meta, zonedID.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = baremetalAPI.DeleteServer(&baremetal.DeleteServerRequest{
		Zone:     zonedID.Zone,
		ServerID: zonedID.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = waitForBaremetalServer(ctx, baremetalAPI, zonedID.Zone, zonedID.ID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}

func baremetalInstallAttributeMissing(field *baremetal.OSOSField, d *schema.ResourceData, attribute string) bool {
	if field != nil && field.Required && field.DefaultValue == nil {
		if _, attributeExists := d.GetOk(attribute); !attributeExists {
			return true
		}
	}
	return false
}

// validateInstallConfig validates that schema contains attribute required for OS install
func validateInstallConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	baremetalAPI, zone, err := BaremetalAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	os, err := baremetalAPI.GetOS(&baremetal.GetOSRequest{
		Zone: zone,
		OsID: locality.ExpandID(d.Get("os")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	diags := diag.Diagnostics(nil)
	installAttributes := []struct {
		Attribute string
		Field     *baremetal.OSOSField
	}{
		{
			"user",
			os.User,
		},
		{
			"password",
			os.Password,
		},
		{
			"service_user",
			os.ServiceUser,
		},
		{
			"service_password",
			os.ServicePassword,
		},
	}
	for _, installAttr := range installAttributes {
		if baremetalInstallAttributeMissing(installAttr.Field, d, installAttr.Attribute) {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       installAttr.Attribute + " attribute is required",
				Detail:        installAttr.Attribute + " is required for this os",
				AttributePath: cty.GetAttrPath(installAttr.Attribute),
			})
		}
	}
	return diags
}
