package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
	tfredshift "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/redshift"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/redshift/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func resourceAwsRedshiftScheduledAction() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftScheduledActionCreate,
		Read:   resourceAwsRedshiftScheduledActionRead,
		Update: resourceAwsRedshiftScheduledActionUpdate,
		Delete: resourceAwsRedshiftScheduledActionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"end_time": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"iam_role": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"schedule": {
				Type:     schema.TypeString,
				Required: true,
			},
			"start_time": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"target_action": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pause_cluster": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cluster_identifier": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							ExactlyOneOf: []string{
								"target_action.0.pause_cluster",
								"target_action.0.resize_cluster",
								"target_action.0.resume_cluster",
							},
						},
						"resize_cluster": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"classic": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"cluster_identifier": {
										Type:     schema.TypeString,
										Required: true,
									},
									"cluster_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"node_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"number_of_nodes": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
							ExactlyOneOf: []string{
								"target_action.0.pause_cluster",
								"target_action.0.resize_cluster",
								"target_action.0.resume_cluster",
							},
						},
						"resume_cluster": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cluster_identifier": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							ExactlyOneOf: []string{
								"target_action.0.pause_cluster",
								"target_action.0.resize_cluster",
								"target_action.0.resume_cluster",
							},
						},
					},
				},
			},
		},
	}
}

func resourceAwsRedshiftScheduledActionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	name := d.Get("name").(string)
	input := &redshift.CreateScheduledActionInput{
		Enable:              aws.Bool(d.Get("enable").(bool)),
		IamRole:             aws.String(d.Get("iam_role").(string)),
		Schedule:            aws.String(d.Get("schedule").(string)),
		ScheduledActionName: aws.String(name),
		TargetAction:        expandRedshiftScheduledActionType(d.Get("target_action").([]interface{})[0].(map[string]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		input.ScheduledActionDescription = aws.String(v.(string))
	}

	if v, ok := d.GetOk("end_time"); ok {
		t, _ := time.Parse(time.RFC3339, v.(string))

		input.EndTime = aws.Time(t)
	}

	if v, ok := d.GetOk("start_time"); ok {
		t, _ := time.Parse(time.RFC3339, v.(string))

		input.StartTime = aws.Time(t)
	}

	log.Printf("[DEBUG] Creating Redshift Scheduled Action: %s", input)
	outputRaw, err := tfresource.RetryWhen(
		iamwaiter.PropagationTimeout,
		func() (interface{}, error) {
			return conn.CreateScheduledAction(input)
		},
		func(err error) (bool, error) {
			if tfawserr.ErrMessageContains(err, tfredshift.ErrCodeInvalidParameterValue, "The IAM role must delegate access to Amazon Redshift scheduler") {
				return true, err
			}

			return false, err
		},
	)

	if err != nil {
		return fmt.Errorf("error creating Redshift Scheduled Action (%s): %w", name, err)
	}

	d.SetId(aws.StringValue(outputRaw.(*redshift.CreateScheduledActionOutput).ScheduledActionName))

	return resourceAwsRedshiftScheduledActionRead(d, meta)
}

func resourceAwsRedshiftScheduledActionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	scheduledAction, err := finder.ScheduledActionByName(conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Redshift Scheduled Action (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Redshift Scheduled Action (%s): %w", d.Id(), err)
	}

	d.Set("description", scheduledAction.ScheduledActionDescription)
	if aws.StringValue(scheduledAction.State) == redshift.ScheduledActionStateActive {
		d.Set("enable", true)
	} else {
		d.Set("enable", false)
	}
	if scheduledAction.EndTime != nil {
		d.Set("end_time", aws.TimeValue(scheduledAction.EndTime).Format(time.RFC3339))
	} else {
		d.Set("end_time", nil)
	}
	d.Set("iam_role", scheduledAction.IamRole)
	d.Set("name", scheduledAction.ScheduledActionName)
	d.Set("schedule", scheduledAction.Schedule)
	if scheduledAction.StartTime != nil {
		d.Set("start_time", aws.TimeValue(scheduledAction.StartTime).Format(time.RFC3339))
	} else {
		d.Set("start_time", nil)
	}

	if scheduledAction.TargetAction != nil {
		if err := d.Set("target_action", []interface{}{flattenRedshiftScheduledActionType(scheduledAction.TargetAction)}); err != nil {
			return fmt.Errorf("error setting target_action: %w", err)
		}
	} else {
		d.Set("target_action", nil)
	}

	return nil
}

func resourceAwsRedshiftScheduledActionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	input := &redshift.ModifyScheduledActionInput{
		ScheduledActionName: aws.String(d.Get("name").(string)),
	}

	if d.HasChange("description") {
		input.ScheduledActionDescription = aws.String(d.Get("description").(string))
	}

	if d.HasChange("enable") {
		input.Enable = aws.Bool(d.Get("enable").(bool))
	}

	if hasChange, v := d.HasChange("end_time"), d.Get("end_time").(string); hasChange && v != "" {
		t, _ := time.Parse(time.RFC3339, v)

		input.EndTime = aws.Time(t)
	}

	if d.HasChange("iam_role") {
		input.IamRole = aws.String(d.Get("iam_role").(string))
	}

	if d.HasChange("schedule") {
		input.Schedule = aws.String(d.Get("schedule").(string))
	}

	if hasChange, v := d.HasChange("start_time"), d.Get("start_time").(string); hasChange && v != "" {
		t, _ := time.Parse(time.RFC3339, v)

		input.StartTime = aws.Time(t)
	}

	if d.HasChange("target_action") {
		input.TargetAction = expandRedshiftScheduledActionType(d.Get("target_action").([]interface{})[0].(map[string]interface{}))
	}

	log.Printf("[DEBUG] Updating Redshift Scheduled Action: %s", input)
	_, err := conn.ModifyScheduledAction(input)

	if err != nil {
		return fmt.Errorf("error updating Redshift Scheduled Action (%s): %w", d.Id(), err)
	}

	return nil
}

func resourceAwsRedshiftScheduledActionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	log.Printf("[DEBUG] Deleting Redshift Scheduled Action: %s", d.Id())
	_, err := conn.DeleteScheduledAction(&redshift.DeleteScheduledActionInput{
		ScheduledActionName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, redshift.ErrCodeScheduledActionNotFoundFault) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Redshift Scheduled Action (%s): %w", d.Id(), err)
	}

	return nil
}

func expandRedshiftScheduledActionType(tfMap map[string]interface{}) *redshift.ScheduledActionType {
	if tfMap == nil {
		return nil
	}

	apiObject := &redshift.ScheduledActionType{}

	if v, ok := tfMap["pause_cluster"].([]interface{}); ok && len(v) > 0 {
		apiObject.PauseCluster = expandRedshiftPauseClusterMessage(v[0].(map[string]interface{}))
	}

	if v, ok := tfMap["resize_cluster"].([]interface{}); ok && len(v) > 0 {
		apiObject.ResizeCluster = expandRedshiftResizeClusterMessage(v[0].(map[string]interface{}))
	}

	if v, ok := tfMap["resume_cluster"].([]interface{}); ok && len(v) > 0 {
		apiObject.ResumeCluster = expandRedshiftResumeClusterMessage(v[0].(map[string]interface{}))
	}

	return apiObject
}

func expandRedshiftPauseClusterMessage(tfMap map[string]interface{}) *redshift.PauseClusterMessage {
	if tfMap == nil {
		return nil
	}

	apiObject := &redshift.PauseClusterMessage{}

	if v, ok := tfMap["cluster_identifier"].(string); ok && v != "" {
		apiObject.ClusterIdentifier = aws.String(v)
	}

	return apiObject
}

func expandRedshiftResizeClusterMessage(tfMap map[string]interface{}) *redshift.ResizeClusterMessage {
	if tfMap == nil {
		return nil
	}

	apiObject := &redshift.ResizeClusterMessage{}

	if v, ok := tfMap["classic"].(bool); ok {
		apiObject.Classic = aws.Bool(v)
	}

	if v, ok := tfMap["cluster_identifier"].(string); ok && v != "" {
		apiObject.ClusterIdentifier = aws.String(v)
	}

	if v, ok := tfMap["cluster_type"].(string); ok && v != "" {
		apiObject.ClusterType = aws.String(v)
	}

	if v, ok := tfMap["node_type"].(string); ok && v != "" {
		apiObject.NodeType = aws.String(v)
	}

	if v, ok := tfMap["number_of_nodes"].(int); ok && v != 0 {
		apiObject.NumberOfNodes = aws.Int64(int64(v))
	}

	return apiObject
}

func expandRedshiftResumeClusterMessage(tfMap map[string]interface{}) *redshift.ResumeClusterMessage {
	if tfMap == nil {
		return nil
	}

	apiObject := &redshift.ResumeClusterMessage{}

	if v, ok := tfMap["cluster_identifier"].(string); ok && v != "" {
		apiObject.ClusterIdentifier = aws.String(v)
	}

	return apiObject
}

func flattenRedshiftScheduledActionType(apiObject *redshift.ScheduledActionType) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.PauseCluster; v != nil {
		tfMap["pause_cluster"] = []interface{}{flattenRedshiftPauseClusterMessage(v)}
	}

	if v := apiObject.ResizeCluster; v != nil {
		tfMap["resize_cluster"] = []interface{}{flattenRedshiftResizeClusterMessage(v)}
	}

	if v := apiObject.ResumeCluster; v != nil {
		tfMap["resume_cluster"] = []interface{}{flattenRedshiftResumeClusterMessage(v)}
	}

	return tfMap
}

func flattenRedshiftPauseClusterMessage(apiObject *redshift.PauseClusterMessage) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.ClusterIdentifier; v != nil {
		tfMap["cluster_identifier"] = aws.StringValue(v)
	}

	return tfMap
}

func flattenRedshiftResizeClusterMessage(apiObject *redshift.ResizeClusterMessage) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.Classic; v != nil {
		tfMap["classic"] = aws.BoolValue(v)
	}

	if v := apiObject.ClusterIdentifier; v != nil {
		tfMap["cluster_identifier"] = aws.StringValue(v)
	}

	if v := apiObject.ClusterType; v != nil {
		tfMap["cluster_type"] = aws.StringValue(v)
	}

	if v := apiObject.NodeType; v != nil {
		tfMap["node_type"] = aws.StringValue(v)
	}

	if v := apiObject.NumberOfNodes; v != nil {
		tfMap["number_of_nodes"] = aws.Int64Value(v)
	}

	return tfMap
}

func flattenRedshiftResumeClusterMessage(apiObject *redshift.ResumeClusterMessage) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.ClusterIdentifier; v != nil {
		tfMap["cluster_identifier"] = aws.StringValue(v)
	}

	return tfMap
}
