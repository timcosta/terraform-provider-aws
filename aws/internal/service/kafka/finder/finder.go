package finder

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func ClusterByARN(conn *kafka.Kafka, arn string) (*kafka.ClusterInfo, error) {
	input := &kafka.DescribeClusterInput{
		ClusterArn: aws.String(arn),
	}

	output, err := conn.DescribeCluster(input)

	if tfawserr.ErrCodeEquals(err, kafka.ErrCodeNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.ClusterInfo == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.ClusterInfo, nil
}

func ClusterOperationByARN(conn *kafka.Kafka, arn string) (*kafka.ClusterOperationInfo, error) {
	input := &kafka.DescribeClusterOperationInput{
		ClusterOperationArn: aws.String(arn),
	}

	output, err := conn.DescribeClusterOperation(input)

	if tfawserr.ErrCodeEquals(err, kafka.ErrCodeNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.ClusterOperationInfo == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.ClusterOperationInfo, nil
}

func ConfigurationByARN(conn *kafka.Kafka, arn string) (*kafka.DescribeConfigurationOutput, error) {
	input := &kafka.DescribeConfigurationInput{
		Arn: aws.String(arn),
	}

	output, err := conn.DescribeConfiguration(input)

	if tfawserr.ErrMessageContains(err, kafka.ErrCodeBadRequestException, "Configuration ARN does not exist") {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}
