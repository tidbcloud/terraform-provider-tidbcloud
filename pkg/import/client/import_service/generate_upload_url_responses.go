// Code generated by go-swagger; DO NOT EDIT.

package import_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"fmt"
	"io"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"

	"github.com/tidbcloud/terraform-provider-tidbcloud/pkg/import/models"
)

// GenerateUploadURLReader is a Reader for the GenerateUploadURL structure.
type GenerateUploadURLReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GenerateUploadURLReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGenerateUploadURLOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGenerateUploadURLDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGenerateUploadURLOK creates a GenerateUploadURLOK with default headers values
func NewGenerateUploadURLOK() *GenerateUploadURLOK {
	return &GenerateUploadURLOK{}
}

/*
GenerateUploadURLOK describes a response with status code 200, with default header values.

A successful response.
*/
type GenerateUploadURLOK struct {
	Payload *models.OpenapiGenerateUploadURLResq
}

// IsSuccess returns true when this generate upload Url o k response has a 2xx status code
func (o *GenerateUploadURLOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this generate upload Url o k response has a 3xx status code
func (o *GenerateUploadURLOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this generate upload Url o k response has a 4xx status code
func (o *GenerateUploadURLOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this generate upload Url o k response has a 5xx status code
func (o *GenerateUploadURLOK) IsServerError() bool {
	return false
}

// IsCode returns true when this generate upload Url o k response a status code equal to that given
func (o *GenerateUploadURLOK) IsCode(code int) bool {
	return code == 200
}

func (o *GenerateUploadURLOK) Error() string {
	return fmt.Sprintf("[POST /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url][%d] generateUploadUrlOK  %+v", 200, o.Payload)
}

func (o *GenerateUploadURLOK) String() string {
	return fmt.Sprintf("[POST /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url][%d] generateUploadUrlOK  %+v", 200, o.Payload)
}

func (o *GenerateUploadURLOK) GetPayload() *models.OpenapiGenerateUploadURLResq {
	return o.Payload
}

func (o *GenerateUploadURLOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.OpenapiGenerateUploadURLResq)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGenerateUploadURLDefault creates a GenerateUploadURLDefault with default headers values
func NewGenerateUploadURLDefault(code int) *GenerateUploadURLDefault {
	return &GenerateUploadURLDefault{
		_statusCode: code,
	}
}

/*
GenerateUploadURLDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type GenerateUploadURLDefault struct {
	_statusCode int

	Payload *models.GooglerpcStatus
}

// Code gets the status code for the generate upload URL default response
func (o *GenerateUploadURLDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this generate upload URL default response has a 2xx status code
func (o *GenerateUploadURLDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this generate upload URL default response has a 3xx status code
func (o *GenerateUploadURLDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this generate upload URL default response has a 4xx status code
func (o *GenerateUploadURLDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this generate upload URL default response has a 5xx status code
func (o *GenerateUploadURLDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this generate upload URL default response a status code equal to that given
func (o *GenerateUploadURLDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GenerateUploadURLDefault) Error() string {
	return fmt.Sprintf("[POST /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url][%d] GenerateUploadURL default  %+v", o._statusCode, o.Payload)
}

func (o *GenerateUploadURLDefault) String() string {
	return fmt.Sprintf("[POST /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url][%d] GenerateUploadURL default  %+v", o._statusCode, o.Payload)
}

func (o *GenerateUploadURLDefault) GetPayload() *models.GooglerpcStatus {
	return o.Payload
}

func (o *GenerateUploadURLDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
GenerateUploadURLBody generate upload URL body
swagger:model GenerateUploadURLBody
*/
type GenerateUploadURLBody struct {

	// content length
	// Required: true
	ContentLength *string `json:"content_length"`

	// file name
	// Required: true
	FileName *string `json:"file_name"`
}

// Validate validates this generate upload URL body
func (o *GenerateUploadURLBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateContentLength(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateFileName(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GenerateUploadURLBody) validateContentLength(formats strfmt.Registry) error {

	if err := validate.Required("body"+"."+"content_length", "body", o.ContentLength); err != nil {
		return err
	}

	return nil
}

func (o *GenerateUploadURLBody) validateFileName(formats strfmt.Registry) error {

	if err := validate.Required("body"+"."+"file_name", "body", o.FileName); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this generate upload URL body based on context it is used
func (o *GenerateUploadURLBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GenerateUploadURLBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GenerateUploadURLBody) UnmarshalBinary(b []byte) error {
	var res GenerateUploadURLBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
