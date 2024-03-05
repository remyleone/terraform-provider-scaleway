package scaleway

import (
	"context"
	"fmt"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayMNQSQSQueue() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayMNQSQSQueueCreate,
		ReadContext:   resourceScalewayMNQSQSQueueRead,
		UpdateContext: resourceScalewayMNQSQSQueueUpdate,
		DeleteContext: resourceScalewayMNQSQSQueueDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultMNQQueueTimeout),
			Update:  schema.DefaultTimeout(defaultMNQQueueTimeout),
			Delete:  schema.DefaultTimeout(defaultMNQQueueTimeout),
			Default: schema.DefaultTimeout(defaultMNQQueueTimeout),
		}, SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				Description:   "The name of the queue. Conflicts with name_prefix.",
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				Description:   "Creates a unique name beginning with the specified prefix. Conflicts with name.",
				ConflictsWith: []string{"name"},
			},
			"sqs_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://sqs.mnq.{region}.scaleway.com",
				Description: "The sqs endpoint",
			},
			"access_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "SQS access key",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "SQS secret key",
			},
			"fifo_queue": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether the queue is a FIFO queue. If true, the queue name must end with .fifo",
			},
			"content_based_deduplication": {
				Type:        schema.TypeBool,
				Computed:    true,
				Optional:    true,
				Description: "Specifies whether to enable content-based deduplication. Allows omitting the deduplication ID",
			},
			"receive_wait_time_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     DefaultQueueReceiveMessageWaitTimeSeconds,
				Description: "The number of seconds to wait for a message to arrive in the queue before returning.",
			},
			"visibility_timeout_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      DefaultQueueVisibilityTimeout,
				ValidateFunc: validation.IntBetween(0, 43_200),
				Description:  "The number of seconds a message is hidden from other consumers.",
			},
			"message_max_age": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      DefaultQueueMessageRetentionPeriod,
				ValidateFunc: validation.IntBetween(60, 1_209_600),
				Description:  "The number of seconds the queue retains a message.",
			},
			"message_max_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      DefaultQueueMaximumMessageSize,
				ValidateFunc: validation.IntBetween(1024, 262_144),
				Description:  "The maximum size of a message. Should be in bytes.",
			},
			"region":     regionSchema(),
			"project_id": project.ProjectIDSchema(),

			// Computed

			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL of the queue",
			},
		},
		CustomizeDiff: resourceMNQQueueCustomizeDiff,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceMNQSQSQueueResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceMNQSQSQueueStateUpgradeV0,
			},
		},
	}
}

func resourceScalewayMNQSQSQueueCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQSQSAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	projectID, _, err := extractProjectID(d, meta.(*meta2.Meta))
	if err != nil {
		return diag.FromErr(err)
	}

	sqsInfo, err := api.GetSqsInfo(&mnq.SqsAPIGetSqsInfoRequest{
		Region:    region,
		ProjectID: projectID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if sqsInfo.Status != mnq.SqsInfoStatusEnabled {
		return diag.FromErr(fmt.Errorf("expected sqs to be enabled for given project, got: %q", sqsInfo.Status))
	}

	sqsClient, _, err := SQSClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	isFifo := d.Get("fifo_queue").(bool)
	queueName := resourceMNQQueueName(d.Get("name"), d.Get("name_prefix"), true, isFifo)

	attributes, err := awsResourceDataToAttributes(d, resourceScalewayMNQSQSQueue().Schema, SQSAttributesToResourceMap)
	if err != nil {
		return diag.FromErr(err)
	}

	input := &sqs.CreateQueueInput{
		Attributes: attributes,
		QueueName:  scw.StringPtr(queueName),
	}

	_, err = retryWhenAWSErrCodeEquals(ctx, []string{sqs.ErrCodeQueueDeletedRecently}, &RetryWhenConfig[*sqs.CreateQueueOutput]{
		Timeout:  d.Timeout(schema.TimeoutCreate),
		Interval: defaultMNQQueueRetryInterval,
		Function: func() (*sqs.CreateQueueOutput, error) {
			return sqsClient.CreateQueueWithContext(ctx, input)
		},
	})
	if err != nil {
		return diag.Errorf("failed to create SQS Queue: %s", err)
	}

	d.SetId(composeMNQID(region, projectID, queueName))

	return resourceScalewayMNQSQSQueueRead(ctx, d, meta)
}

func resourceScalewayMNQSQSQueueRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sqsClient, _, err := SQSClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	region, projectID, queueName, err := decomposeMNQID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	queue, err := retryWhenAWSErrCodeEquals(ctx, []string{sqs.ErrCodeQueueDoesNotExist}, &RetryWhenConfig[*sqs.GetQueueUrlOutput]{
		Timeout:  d.Timeout(schema.TimeoutRead),
		Interval: defaultMNQQueueRetryInterval,
		Function: func() (*sqs.GetQueueUrlOutput, error) {
			return sqsClient.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
				QueueName: aws.String(queueName),
			})
		},
	})
	if err != nil {
		return diag.Errorf("failed to get the SQS Queue URL: %s", err)
	}

	queueAttributes, err := sqsClient.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       queue.QueueUrl,
		AttributeNames: getSQSAttributeNames(),
	})
	if err != nil {
		return diag.Errorf("failed to get the SQS Queue attributes: %s", err)
	}

	values, err := awsAttributesToResourceData(queueAttributes.Attributes, resourceScalewayMNQSQSQueue().Schema, SQSAttributesToResourceMap)
	if err != nil {
		return diag.Errorf("failed to convert SQS Queue attributes to resource data: %s", err)
	}

	_ = d.Set("name", queueName)
	_ = d.Set("region", region)
	_ = d.Set("project_id", projectID)
	_ = d.Set("url", flattenStringPtr(queue.QueueUrl))

	for k, v := range values {
		_ = d.Set(k, v) // lintignore: R001
	}

	return nil
}

func resourceScalewayMNQSQSQueueUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sqsClient, _, err := SQSClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, _, queueName, err := decomposeMNQID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	queue, err := retryWhenAWSErrCodeEquals(ctx, []string{sqs.ErrCodeQueueDoesNotExist}, &RetryWhenConfig[*sqs.GetQueueUrlOutput]{
		Timeout:  d.Timeout(schema.TimeoutUpdate),
		Interval: defaultMNQQueueRetryInterval,
		Function: func() (*sqs.GetQueueUrlOutput, error) {
			return sqsClient.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
				QueueName: aws.String(queueName),
			})
		},
	})
	if err != nil {
		return diag.Errorf("failed to get the SQS Queue URL: %s", err)
	}

	attributes, err := awsResourceDataToAttributes(d, resourceScalewayMNQSQSQueue().Schema, SQSAttributesToResourceMap)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = sqsClient.SetQueueAttributesWithContext(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl:   queue.QueueUrl,
		Attributes: attributes,
	})
	if err != nil {
		return diag.Errorf("failed to update SQS Queue attributes: %s", err)
	}

	return resourceScalewayMNQSQSQueueRead(ctx, d, meta)
}

func resourceScalewayMNQSQSQueueDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sqsClient, _, err := SQSClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, _, queueName, err := decomposeMNQID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	queue, err := sqsClient.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		if tfawserr.ErrCodeEquals(err, sqs.ErrCodeQueueDoesNotExist) {
			return nil
		}

		return diag.Errorf("failed to get the SQS Queue URL: %s", err)
	}

	_, err = sqsClient.DeleteQueueWithContext(ctx, &sqs.DeleteQueueInput{
		QueueUrl: queue.QueueUrl,
	})
	if err != nil {
		if tfawserr.ErrCodeEquals(err, sqs.ErrCodeQueueDoesNotExist) {
			return nil
		}

		return diag.Errorf("failed to delete SQS Queue (%s): %s", d.Id(), err)
	}

	_, _ = retryWhenAWSErrCodeNotEquals(ctx, []string{sqs.ErrCodeQueueDoesNotExist}, &RetryWhenConfig[*sqs.GetQueueUrlOutput]{
		Timeout:  d.Timeout(schema.TimeoutCreate),
		Interval: defaultMNQQueueRetryInterval,
		Function: func() (*sqs.GetQueueUrlOutput, error) {
			return sqsClient.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
				QueueName: aws.String(queueName),
			})
		},
	})

	return nil
}
