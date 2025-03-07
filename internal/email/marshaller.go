package email

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func redirectPanic(err error) {
	r := recover()
	if r != nil {
		err = errors.New(fmt.Sprint(r))
	}
}

type emailItemRow struct {
	Id         string                 `dynamodbav:"id"`
	Attributes map[string]interface{} `dynamodbav:"attributes"`
}

func (email emailItemRow) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(email.Id)
	if err != nil {
		panic(err)
	}

	return map[string]types.AttributeValue{"id": id}
}

type emailMarshaller struct{}

func (m *emailMarshaller) itemToModel(item emailItemRow) (Email, error) {
	statusAttribute, statusOk := item.Attributes["status"]
	if !statusOk {
		return Email{}, errors.New("status not set")
	}

	status, statusOk := statusAttribute.(string)
	if !statusOk {
		return Email{}, errors.New("status not a string")
	}

	emlFilepath, emlFilepathOk := item.Attributes["eml_filepath"].(string)
	if !emlFilepathOk {
		return Email{}, errors.New("eml_filepath not a string")
	}

	successCallback, successCallbackOk := item.Attributes["success_callback"].(string)
	if !successCallbackOk {
		return Email{}, errors.New("success_callback not a string")
	}

	failureCallback, failureCallbackOk := item.Attributes["failure_callback"].(string)
	if !failureCallbackOk {
		return Email{}, errors.New("failure_callback not a string")
	}

	return Email{
		Id:              item.Id,
		Status:          status,
		EmlFilePath:     emlFilepath,
		SuccessCallback: successCallback,
		FailureCallback: failureCallback,
	}, nil
}

func (m *emailMarshaller) Marshal(email Email) (map[string]types.AttributeValue, error) {
	return attributevalue.MarshalMap(
		emailItemRow{
			Id: email.Id,
			Attributes: map[string]interface{}{
				"status":           email.Status,
				"eml_filepath":     email.EmlFilePath,
				"success_callback": email.SuccessCallback,
				"failure_callback": email.FailureCallback,
			},
		},
	)
}

func (m *emailMarshaller) UnmarshalListOfMaps(attrsList []map[string]types.AttributeValue) (emails []Email, err error) {
	defer redirectPanic(err)

	var items []emailItemRow
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Email{}, err
	}

	for _, item := range items {
		email, err := m.itemToModel(item)
		if err != nil {
			return []Email{}, err
		}

		emails = append(emails, email)
	}

	return emails, nil
}
