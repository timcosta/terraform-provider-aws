package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	tfeks "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/eks"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/eks/finder"
)

func dataSourceAwsEksNodeGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAwsEksNodeGroupRead,

		Schema: map[string]*schema.Schema{
			"ami_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"disk_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"instance_types": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"labels": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"node_group_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"node_role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"release_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"remote_access": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ec2_ssh_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"source_security_group_ids": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"resources": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"autoscaling_groups": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"remote_access_security_group_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"scaling_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"desired_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"min_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsSchemaComputed(),
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEksNodeGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*AWSClient).eksconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	clusterName := d.Get("cluster_name").(string)
	nodeGroupName := d.Get("node_group_name").(string)
	id := tfeks.NodeGroupCreateResourceID(clusterName, nodeGroupName)
	nodeGroup, err := finder.NodegroupByClusterNameAndNodegroupName(conn, clusterName, nodeGroupName)

	if err != nil {
		return diag.Errorf("error reading EKS Node Group (%s): %s", id, err)
	}

	d.SetId(id)

	d.Set("ami_type", nodeGroup.AmiType)
	d.Set("arn", nodeGroup.NodegroupArn)
	d.Set("cluster_name", nodeGroup.ClusterName)
	d.Set("disk_size", nodeGroup.DiskSize)
	d.Set("instance_types", nodeGroup.InstanceTypes)
	d.Set("labels", nodeGroup.Labels)
	d.Set("node_group_name", nodeGroup.NodegroupName)
	d.Set("node_role_arn", nodeGroup.NodeRole)
	d.Set("release_version", nodeGroup.ReleaseVersion)

	if err := d.Set("remote_access", flattenEksRemoteAccessConfig(nodeGroup.RemoteAccess)); err != nil {
		return diag.Errorf("error setting remote_access: %s", err)
	}

	if err := d.Set("resources", flattenEksNodeGroupResources(nodeGroup.Resources)); err != nil {
		return diag.Errorf("error setting resources: %s", err)
	}

	if nodeGroup.ScalingConfig != nil {
		if err := d.Set("scaling_config", []interface{}{flattenEksNodeGroupScalingConfig(nodeGroup.ScalingConfig)}); err != nil {
			return diag.Errorf("error setting scaling_config: %s", err)
		}
	} else {
		d.Set("scaling_config", nil)
	}

	d.Set("status", nodeGroup.Status)

	if err := d.Set("subnet_ids", aws.StringValueSlice(nodeGroup.Subnets)); err != nil {
		return diag.Errorf("error setting subnets: %s", err)
	}

	if err := d.Set("tags", keyvaluetags.EksKeyValueTags(nodeGroup.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return diag.Errorf("error setting tags: %s", err)
	}

	d.Set("version", nodeGroup.Version)

	return nil
}
