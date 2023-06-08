package validation

import (
	"errors"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
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
		err := descriptorEksValidations((*v).descriptor)
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

func descriptorEksValidations(descriptorFile commons.DescriptorFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	err = validateVPCCidr(descriptorFile)
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
