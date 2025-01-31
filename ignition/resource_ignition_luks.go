package ignition

import (
	"encoding/json"
	"fmt"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/coreos/vcontext/path"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// TODO: add http_headers for other ressources too!
func dataSourceLuks() *schema.Resource {
	return &schema.Resource{
		Exists: resourceLuksExists,
		Read:   resourceLuksRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"device": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mime": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "text/plain",
						},

						"content": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"source": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"compression": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"verification": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"http_headers": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"value": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},
			"label": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"uuid": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"options": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"wipe_volume": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"clevis": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tang": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
									"thumbprint": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
						"tpm2": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"treshold": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"custom": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pin": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
									"config": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
									"needs_network": {
										Type:     schema.TypeBool,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},
			"rendered": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLuksRead(d *schema.ResourceData, meta interface{}) error {
	id, err := buildLuks(d)
	if err != nil {
		return err
	}

	d.SetId(id)
	return nil
}

func resourceLuksExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	id, err := buildLuks(d)
	if err != nil {
		return false, err
	}

	return id == d.Id(), nil
}

func buildLuks(d *schema.ResourceData) (string, error) {
	luks := &types.Luks{
		Name: d.Get("name").(string),
	}

	device, hasDevice := d.GetOk("device")
	if hasDevice {
		sdevice := device.(string)
		luks.Device = &sdevice
	}

	label, hasLabel := d.GetOk("label")
	if hasLabel {
		slabel := label.(string)
		luks.Label = &slabel
	}

	uuid, hasUUID := d.GetOk("uuid")
	if hasUUID {
		suuid := uuid.(string)
		luks.UUID = &suuid
	}

	wipeVol, hasWipeVol := d.GetOk("wipe_volume")
	if hasWipeVol {
		bwipeVol := wipeVol.(bool)
		luks.WipeVolume = &bwipeVol
	}

	_, hasContent := d.GetOk("content")
	_, hasSource := d.GetOk("source")

	if hasContent && hasSource {
		return "", fmt.Errorf("content and source options are incompatible")
	}

	if hasContent || hasSource {
		var keyFile types.Resource
		if hasContent {
			s := encodeDataURL(
				d.Get("content.0.mime").(string),
				d.Get("content.0.content").(string),
			)
			keyFile.Source = &s
		}

		if hasSource {
			src := d.Get("source.0.source").(string)
			if src != "" {
				keyFile.Source = &src
			}
			compression := d.Get("source.0.compression").(string)
			if compression != "" {
				keyFile.Compression = &compression
			}
			h := d.Get("source.0.verification").(string)
			if h != "" {
				keyFile.Verification.Hash = &h
			}
			for _, raw := range d.Get("source.0.http_headers").([]interface{}) {
				v := raw.(map[string]interface{})
				p := types.HTTPHeader{
					Name: v["name"].(string),
				}

				value := v["value"]
				if value != nil {
					svalue := value.(string)
					p.Value = &svalue
				}

				keyFile.HTTPHeaders = append(keyFile.HTTPHeaders, p)
			}
		}

		luks.KeyFile = keyFile
	}

	_, hasClevis := d.GetOk("clevis")
	if hasClevis {
		var clevis types.Clevis

		tpm2, hasTpm2 := d.GetOk("clevis.0.tpm2")
		if hasTpm2 {
			btpm2 := tpm2.(bool)
			clevis.Tpm2 = &btpm2
		}

		threshold, hasThreshold := d.GetOk("clevis.0.threshold")
		if hasThreshold {
			ithreshold := threshold.(int)
			clevis.Threshold = &ithreshold
		}

		for _, raw := range d.Get("clevis.0.tang").([]interface{}) {
			v := raw.(map[string]interface{})
			p := types.Tang{
				URL: v["url"].(string),
			}

			thumbprint, hasThumbprint := d.GetOk("clevis.0.tang.thumbprint")
			if hasThumbprint {
				sthumbprint := thumbprint.(string)
				p.Thumbprint = &sthumbprint
			}

			clevis.Tang = append(clevis.Tang, p)
		}

		_, hasCustom := d.GetOk("clevis.0.custom")
		if hasCustom {
			custom := types.Custom{
				Pin:    d.Get("clevis.0.pin").(string),
				Config: d.Get("clevis.0.config").(string),
			}

			needsNetwork, hasNeedsNetwork := d.GetOk("clevis.0.needs_network")
			if hasNeedsNetwork {
				bneedsNetwork := needsNetwork.(bool)
				custom.NeedsNetwork = &bneedsNetwork
			}

			clevis.Custom = &custom
		}

		luks.Clevis = &clevis
	}

	for _, value := range d.Get("options").([]interface{}) {
		luks.Options = append(luks.Options, value.(types.LuksOption))
	}

	if err := handleReport(luks.Validate(path.ContextPath{})); err != nil {
		return "", err
	}

	b, err := json.Marshal(luks)
	if err != nil {
		return "", err
	}
	err = d.Set("rendered", string(b))
	if err != nil {
		return "", err
	}

	return hash(string(b)), nil
}
