//nolint:lll
package store

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

const (
	// AccountDefaultSSMAliasKeyID represents account's default SSM alias used
	// to encrypt/decrypt configuration secrets
	AccountDefaultSSMAliasKeyID = "aws/ssm"
)

// validPathKeyFormat is the format that is expected for key names inside
// parameter store when using paths
var validPathKeyFormat = regexp.MustCompile(`^(\/[\w\-\.]+)+$`)

// SSMStore implements the Store interface for storing configurations in
// SSM Parameter Store
type SSMStore struct {
	svc ssmiface.SSMAPI
}

// NewSSMStore creates a new SSMStore
func NewSSMStore(numRetries int) (*SSMStore, error) {
	ssmSession, region, err := getSession(numRetries)
	if err != nil {
		return nil, err
	}

	svc := ssm.New(ssmSession, &aws.Config{
		MaxRetries: aws.Int(numRetries),
		Region:     region,
	})

	return &SSMStore{
		svc: svc,
	}, nil
}

func (s *SSMStore) KMSKey() string {
	return fmt.Sprintf("alias/%s", AccountDefaultSSMAliasKeyID)
}

// Put adds a given value to a the system identified by name.
// If the configuration already exists, then it writes a new version.
func (s *SSMStore) Put(name ParameterName, value Value) error {
	version := 1

	// first read to get the current version
	current, err := s.Get(name, -1)
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	//nolint:gomnd
	if err == nil {
		version = current.Meta.Version + 1
	}

	putParameterInput := &ssm.PutParameterInput{
		Name:        aws.String(s.parameterNameToString(name)),
		Type:        aws.String("String"),
		Value:       value.Value,
		Overwrite:   aws.Bool(true),
		Description: aws.String(strconv.Itoa(version)),
	}

	if value.Meta.Secure {
		putParameterInput.Type = aws.String("SecureString")
		putParameterInput.KeyId = aws.String(s.KMSKey())
	}

	// This API call returns an empty struct
	_, err = s.svc.PutParameter(putParameterInput)
	if err != nil {
		return err
	}

	return nil
}

// Get reads a configuration from the parameter store at a specific version.
// To grab the latest version, use -1 as the version number.
func (s *SSMStore) Get(name ParameterName, version int) (Value, error) {
	if version == -1 {
		return s.getLatest(name)
	}

	return s.getVersion(name, version)
}

//nolint:funlen
func (s *SSMStore) List(prefix string, includeValues bool) ([]Value, error) {
	configs := map[string]Value{}

	describeParametersInput := &ssm.DescribeParametersInput{
		ParameterFilters: []*ssm.ParameterStringFilter{
			{
				Key:    aws.String("Path"),
				Option: aws.String("Recursive"),
				Values: []*string{aws.String(path.Join("/", prefix))},
			},
		},
	}

	err := s.svc.DescribeParametersPages(describeParametersInput, func(resp *ssm.DescribeParametersOutput, lastPage bool) bool {
		for _, meta := range resp.Parameters {
			if !s.validateName(*meta.Name) {
				continue
			}

			configMeta := parameterMetaToValueMeta(meta)

			configs[configMeta.Key] = Value{
				Value: nil,
				Meta:  configMeta,
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	if includeValues {
		valueKeys := keys(configs)
		batchSize := 10

		for i := 0; i < len(valueKeys); i += batchSize {
			batchEnd := i + batchSize
			if i+batchSize > len(valueKeys) {
				batchEnd = len(valueKeys)
			}

			batch := valueKeys[i:batchEnd]

			getParametersInput := &ssm.GetParametersInput{
				Names:          stringsToAWSStrings(batch),
				WithDecryption: aws.Bool(true),
			}

			resp, err := s.svc.GetParameters(getParametersInput)
			if err != nil {
				return nil, err
			}

			for _, param := range resp.Parameters {
				value := configs[*param.Name]
				value.Value = param.Value
				configs[*param.Name] = value
			}
		}
	}

	return values(configs), nil
}

// ListRaw lists all configuration keys and values for a given prefix.
// Does not include any other meta-data. Uses faster AWS APIs with much higher
// rate-limits.
// Suitable for use in production environments.
func (s *SSMStore) ListRaw(prefix string) ([]RawValue, error) {
	values := map[string]RawValue{}

	getParametersByPathInput := &ssm.GetParametersByPathInput{
		Path:           aws.String(path.Join("/", prefix)),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	err := s.svc.GetParametersByPathPages(getParametersByPathInput, func(resp *ssm.GetParametersByPathOutput, lastPage bool) bool {
		for _, param := range resp.Parameters {
			if !s.validateName(*param.Name) {
				continue
			}

			values[*param.Name] = RawValue{
				Value: *param.Value,
				Key:   *param.Name,
			}
		}

		return !lastPage
	})

	if err != nil {
		// If the error is an access-denied exception
		awsErr, isAwserr := err.(awserr.Error)
		if isAwserr {
			return nil, awsErr
		}

		return nil, err
	}

	rawValues := make([]RawValue, len(values))
	i := 0

	for _, rawValue := range values {
		rawValues[i] = rawValue
		i++
	}

	return rawValues, nil
}

// Delete removes a configuration from the parameter store. Note this removes
// all versions of the configuration.
func (s *SSMStore) Delete(name ParameterName) error {
	// first read to ensure parameter present
	_, err := s.Get(name, -1)
	if err != nil {
		return err
	}

	deleteParameterInput := &ssm.DeleteParameterInput{
		Name: aws.String(s.parameterNameToString(name)),
	}

	_, err = s.svc.DeleteParameter(deleteParameterInput)
	if err != nil {
		return err
	}

	return nil
}

func (s *SSMStore) parameterNameToString(name ParameterName) string {
	return path.Join([]string{"/", name.ParameterPath, name.Name}...)
}

func (s *SSMStore) getVersion(name ParameterName, version int) (Value, error) {
	getParameterHistoryInput := &ssm.GetParameterHistoryInput{
		Name:           aws.String(s.parameterNameToString(name)),
		WithDecryption: aws.Bool(true),
	}

	var result Value

	if err := s.svc.GetParameterHistoryPages(getParameterHistoryInput, func(o *ssm.GetParameterHistoryOutput, lastPage bool) bool {
		for _, history := range o.Parameters {
			thisVersion := 0
			if history.Description != nil {
				thisVersion, _ = strconv.Atoi(*history.Description)
			}
			if thisVersion == version {
				result = Value{
					Value: history.Value,
					Meta: Metadata{
						Key:              *history.Name,
						Description:      *history.Description,
						Secure:           (*history.Type == "SecureString"),
						Version:          thisVersion,
						LastModifiedDate: *history.LastModifiedDate,
						LastModifiedUser: *history.LastModifiedUser,
					},
				}

				return false
			}
		}
		return true
	}); err != nil {
		return Value{}, ErrConfigNotFound
	}

	if result.Value != nil {
		return result, nil
	}

	return Value{}, ErrConfigNotFound
}

func (s *SSMStore) getLatest(name ParameterName) (Value, error) {
	parameterNameString := s.parameterNameToString(name)

	getParametersInput := &ssm.GetParametersInput{
		Names:          []*string{aws.String(parameterNameString)},
		WithDecryption: aws.Bool(true),
	}

	resp, err := s.svc.GetParameters(getParametersInput)
	if err != nil {
		return Value{}, err
	}

	if len(resp.Parameters) == 0 {
		return Value{}, ErrConfigNotFound
	}

	param := resp.Parameters[0]

	var parameter *ssm.ParameterMetadata

	// To get metadata, we need to use describe parameters

	// There is no way to use describe parameters to get a single key
	// when that key uses paths, so instead get all the keys for a path,
	// then find the one you are looking for
	describeParametersInput := &ssm.DescribeParametersInput{
		ParameterFilters: []*ssm.ParameterStringFilter{
			{
				Key:    aws.String("Path"),
				Option: aws.String("OneLevel"),
				Values: []*string{aws.String(path.Dir(parameterNameString))},
			},
		},
	}

	if err := s.svc.DescribeParametersPages(describeParametersInput, func(o *ssm.DescribeParametersOutput, lastPage bool) bool {
		for _, param := range o.Parameters {
			if *param.Name == parameterNameString {
				parameter = param
				return false
			}
		}
		return true
	}); err != nil {
		return Value{}, err
	}

	if parameter == nil {
		return Value{}, ErrConfigNotFound
	}

	configMeta := parameterMetaToValueMeta(parameter)

	return Value{
		Value: param.Value,
		Meta:  configMeta,
	}, nil
}

func (s *SSMStore) validateName(name string) bool {
	return validPathKeyFormat.MatchString(name)
}

func parameterMetaToValueMeta(p *ssm.ParameterMetadata) Metadata {
	version := 0

	if p.Description != nil {
		version, _ = strconv.Atoi(*p.Description)
	}

	return Metadata{
		Key:              *p.Name,
		Description:      *p.Description,
		Secure:           (*p.Type == "SecureString"),
		Version:          version,
		LastModifiedDate: *p.LastModifiedDate,
		LastModifiedUser: *p.LastModifiedUser,
	}
}

func keys(m map[string]Value) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func values(m map[string]Value) []Value {
	values := []Value{}
	for _, v := range m {
		values = append(values, v)
	}

	return values
}

// Check the interfaces are satisfied
var (
	_ Store = &SSMStore{}
)
