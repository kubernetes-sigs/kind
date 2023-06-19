package validation

import (
	"errors"
	"net"
	"os"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
)

var eksInstance *EKSValidator

const (
	cidrSizeMax = 65536
	cidrSizeMin = 16
)

type EKSValidator struct {
	commonValidator
}

func newEKSValidator() *EKSValidator {
	if eksInstance == nil {
		eksInstance = new(EKSValidator)
	}
	return eksInstance
}

func (v *EKSValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *EKSValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *EKSValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := descriptorEksValidations((*v).descriptor, (*v).secrets)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsEksValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func (v *EKSValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func descriptorEksValidations(descriptorFile commons.DescriptorFile, secretsFile commons.SecretsFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	err = validateVPCCidr(descriptorFile)
	if err != nil {
		return err
	}
	err = eksAZValidation(descriptorFile, secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func secretsEksValidations(secretsFile commons.SecretsFile) error {
	err := commonsSecretsValidations(secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func validateVPCCidr(descriptorFile commons.DescriptorFile) error {
	if descriptorFile.Networks.PodsCidrBlock != "" {
		_, validRange1, _ := net.ParseCIDR("100.64.0.0/10")
		_, validRange2, _ := net.ParseCIDR("198.19.0.0/16")

		_, ipv4Net, _ := net.ParseCIDR(descriptorFile.Networks.PodsCidrBlock)

		cidrSize := cidr.AddressCount(ipv4Net)
		if cidrSize > cidrSizeMax || cidrSize < cidrSizeMin {
			return errors.New("Invalid parameter PodsCidrBlock, CIDR block sizes must be between a /16 netmask and /28 netmask")
		}

		start, end := cidr.AddressRange(ipv4Net)
		if (!validRange1.Contains(start) || !validRange1.Contains(end)) && (!validRange2.Contains(start) || !validRange2.Contains(end)) {
			return errors.New("Invalid parameter PodsCidrBlock, CIDR must be within the 100.64.0.0/10 or 198.19.0.0/16 range")
		}
	}
	return nil
}

func eksAZValidation(descriptorFile commons.DescriptorFile, secretsFile commons.SecretsFile) error {
	awsCredentials := []string{
		"AWS_REGION=" + descriptorFile.Region,
		"AWS_ACCESS_KEY_ID=" + secretsFile.Secrets.AWS.Credentials.AccessKey,
		"AWS_SECRET_ACCESS_KEY=" + secretsFile.Secrets.AWS.Credentials.SecretKey,
	}
	for _, cred := range awsCredentials {
		c := strings.Split(cred, "=")
		envVar := c[0]
		envValue := c[1]
		os.Setenv(envVar, envValue)
	}

	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return err
	}
	svc := ec2.New(sess)
	if descriptorFile.Networks.Subnets != nil {
		privateAZs := []string{}
		for _, subnet := range descriptorFile.Networks.Subnets {
			privateSubnetID, _ := filterPrivateSubnet(svc, &subnet.SubnetId)
			if len(privateSubnetID) > 0 {
				sid := &ec2.DescribeSubnetsInput{
					SubnetIds: []*string{&subnet.SubnetId},
				}
				ds, err := svc.DescribeSubnets(sid)
				if err != nil {
					return err
				}
				for _, describeSubnet := range ds.Subnets {
					if !slices.Contains(privateAZs, *describeSubnet.AvailabilityZone) {
						privateAZs = append(privateAZs, *describeSubnet.AvailabilityZone)
					}
				}
			}
		}
		if len(privateAZs) < 3 {
			return errors.New("Insufficient Availability Zones in region " + descriptorFile.Region + ". Please add at least 3 private subnets in different Availability Zones")
		}
		for _, node := range descriptorFile.WorkerNodes {
			if node.ZoneDistribution == "unbalanced" && node.AZ != "" {
				if !slices.Contains(privateAZs, node.AZ) {
					return errors.New("Worker node " + node.Name + " whose AZ is defined in " + node.AZ + " must match with the AZs associated to the defined subnets in descriptor")
				}
			}
		}
	} else {
		result, err := svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
		if err != nil {
			return err
		}
		if len(result.AvailabilityZones) < 3 {
			return errors.New("Insufficient Availability Zones in region " + descriptorFile.Region + ". Must have at least 3")
		}
		azs := make([]string, 3)
		for i, az := range result.AvailabilityZones {
			if i == 3 {
				break
			}
			azs[i] = *az.ZoneName
		}
		for _, node := range descriptorFile.WorkerNodes {
			if node.ZoneDistribution == "unbalanced" && node.AZ != "" {
				if !slices.Contains(azs, node.AZ) {
					return errors.New("Worker node " + node.Name + " whose AZ is defined in " + node.AZ + " must match with the first three AZs in region " + descriptorFile.Region)
				}
			}
		}
	}
	return nil
}

func filterPrivateSubnet(svc *ec2.EC2, subnetID *string) (string, error) {
	keyname := "association.subnet-id"
	filters := make([]*ec2.Filter, 0)
	filter := ec2.Filter{
		Name: &keyname, Values: []*string{subnetID}}
	filters = append(filters, &filter)

	drti := &ec2.DescribeRouteTablesInput{Filters: filters}
	drto, err := svc.DescribeRouteTables(drti)
	if err != nil {
		return "", err
	}

	var isPublic bool
	for _, associatedRouteTable := range drto.RouteTables {

		for i := range associatedRouteTable.Routes {
			if *associatedRouteTable.Routes[i].DestinationCidrBlock == "0.0.0.0/0" &&
				associatedRouteTable.Routes[i].GatewayId != nil &&
				strings.Contains(*associatedRouteTable.Routes[i].GatewayId, "igw") {
				isPublic = true
			}
		}
	}
	if !isPublic {
		return *subnetID, nil
	} else {
		return "", nil
	}
}
