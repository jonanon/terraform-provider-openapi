package openapi

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/stretchr/testify/assert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateDataSourceSchema(t *testing.T) {

	testCases := []struct {
		name           string
		specSchemaDef  specSchemaDefinition
		expectedResult map[string]*schema.Schema
		expectedError  error
	}{
		{
			name: "happy path -- data source schema contains all schema properties configured as computed",
			specSchemaDef: specSchemaDefinition{
				Properties: specSchemaDefinitionProperties{
					&specSchemaDefinitionProperty{
						Name:     "id",
						Type:     typeString,
						ReadOnly: false,
						Required: true,
					},
					&specSchemaDefinitionProperty{
						Name:     "string_prop",
						Type:     typeString,
						ReadOnly: false,
						Required: true,
					},
					&specSchemaDefinitionProperty{
						Name:     "int_prop",
						Type:     typeInt,
						ReadOnly: false,
						Required: true,
					},
				},
			},
			expectedResult: map[string]*schema.Schema{
				"string_prop": {
					Required: false,
					Optional: true,
					Computed: true,
					Type:     schema.TypeString,
				},
				"int_prop": {
					Required: false,
					Optional: true,
					Computed: true,
					Type:     schema.TypeInt,
				},
			},
			expectedError: nil,
		},
		{
			name:           "sad path -- a terraform schema cannot be created from a specSchemaDefinition ",
			specSchemaDef:  specSchemaDefinition{Properties: specSchemaDefinitionProperties{&specSchemaDefinitionProperty{}}},
			expectedResult: nil,
			expectedError:  errors.New("non supported type "),
		},
	}

	for _, tc := range testCases {
		s, err := tc.specSchemaDef.createDataSourceSchema()
		if tc.expectedError == nil {
			assert.Nil(t, err, tc.name)
			for expectedTerraformPropName, expectedTerraformSchemaProp := range tc.expectedResult {
				assert.Nil(t, s["id"])
				assertDataSourceSchemaProperty(t, s[expectedTerraformPropName], expectedTerraformSchemaProp.Type, tc.name)
			}
		} else {
			assert.Equal(t, tc.expectedError.Error(), err.Error(), tc.name)
		}
	}
}

func TestCreateDataSourceSchema_Subresources(t *testing.T) {
	t.Run("happy path -- data source schema dor sub-resoruce contains all schema properties configured as computed but parent properties", func(t *testing.T) {

		specSchemaDef := specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:             "parent_id",
					Type:             typeString,
					ReadOnly:         false,
					Required:         true,
					IsParentProperty: true,
				},
			},
		}

		// get the Terraform schema which represents a Data Source
		s, err := specSchemaDef.createDataSourceSchema()

		assert.NotNil(t, s)
		assert.NoError(t, err)

		assert.Equal(t, schema.TypeString, s["parent_id"].Type)
		assert.True(t, s["parent_id"].Required)
		assert.False(t, s["parent_id"].Optional)
		assert.False(t, s["parent_id"].Computed)
	})
}

func TestCreateDataSourceSchema_ForNestedObjects(t *testing.T) {
	t.Run("happy path -- a data soruce can be derived from a nested object keeping all the properies attributes as expected", func(t *testing.T) {
		// set up the schema for the nested object
		nestedObjectSchemaDefinition := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				newIntSchemaDefinitionPropertyWithDefaults("origin_port", "", true, false, 80),
				newStringSchemaDefinitionPropertyWithDefaults("protocol", "", true, false, "http"),
			},
		}
		nestedObjectDefault := map[string]interface{}{
			"origin_port": nestedObjectSchemaDefinition.Properties[0].Default,
			"protocol":    nestedObjectSchemaDefinition.Properties[1].Default,
		}
		nestedObject := newObjectSchemaDefinitionPropertyWithDefaults("nested_object", "", true, false, false, nestedObjectDefault, nestedObjectSchemaDefinition)
		propertyWithNestedObjectSchemaDefinition := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				idProperty,
				nestedObject,
			},
		}
		dataValue := map[string]interface{}{
			"id":            propertyWithNestedObjectSchemaDefinition.Properties[0].Default,
			"nested_object": propertyWithNestedObjectSchemaDefinition.Properties[1].Default,
		}
		dataSourceSchema := newObjectSchemaDefinitionPropertyWithDefaults("nested-oobj", "", true, false, false, dataValue, propertyWithNestedObjectSchemaDefinition)
		specSchemaNested := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{dataSourceSchema},
		}

		// get the Terraform schema which represents a Data Source
		s, err := specSchemaNested.createDataSourceSchema()

		assert.NotNil(t, s)
		assert.NoError(t, err)

		// assertions on the properties attributes
		assertDataSourceSchemaProperty(t, s["nested_oobj"], schema.TypeList)

		// 1^ level
		objectResource := s["nested_oobj"].Elem.(*schema.Resource)
		assert.Equal(t, 2, len(objectResource.Schema))

		assertDataSourceSchemaProperty(t, objectResource.Schema["id"], schema.TypeString)
		assertDataSourceSchemaProperty(t, objectResource.Schema["nested_object"], schema.TypeMap)

		// 2^ level
		nestedObjectResource := objectResource.Schema["nested_object"].Elem.(*schema.Resource)
		assert.Equal(t, 2, len(nestedObjectResource.Schema))
		assertDataSourceSchemaProperty(t, nestedObjectResource.Schema["origin_port"], schema.TypeInt)
		assertDataSourceSchemaProperty(t, nestedObjectResource.Schema["protocol"], schema.TypeString)
	})
}

func TestCreateResourceSchema(t *testing.T) {
	Convey("Given a swagger schema definition that has few properties including the id all with terraform compliant names", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:     "string_prop",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the  err returned should not be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the resulted tfResourceSchema should not contain ID property", func() {
				So(tfResourceSchema, ShouldNotContainKey, "id")
			})
			Convey("Then the resulted tfResourceSchema should contain string_prop property", func() {
				So(tfResourceSchema, ShouldContainKey, "string_prop")
			})
		})
	})

	Convey("Given a swagger schema definition that has few properties including the id all with NON terraform compliant names", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "ID",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:     "stringProp",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the  err returned should not be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the resulted tfResourceSchema should not contain ID property", func() {
				So(tfResourceSchema, ShouldNotContainKey, "id")
			})
			Convey("Then the resulted tfResourceSchema should contain a key with the terraform compliant name string_prop ", func() {
				So(tfResourceSchema, ShouldContainKey, "string_prop")
			})
		})
	})

	Convey("Given a swagger schema definition that has few properties including an object property with internal ID field", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:     "string_prop",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:     statusDefaultPropertyName,
					Type:     typeObject,
					ReadOnly: true,
					SpecSchemaDefinition: &specSchemaDefinition{
						Properties: specSchemaDefinitionProperties{
							&specSchemaDefinitionProperty{
								Name:               "id",
								Type:               typeString,
								ReadOnly:           true,
								IsStatusIdentifier: true,
							},
							&specSchemaDefinitionProperty{
								Name:               "name",
								Type:               typeString,
								ReadOnly:           true,
								IsStatusIdentifier: true,
							},
						},
					},
				},
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the  err returned should not be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the resulted tfResourceSchema should not contain ID property", func() {
				So(tfResourceSchema, ShouldNotContainKey, "id")
			})
			Convey("And the resulted tfResourceSchema should contain string_prop property of type string", func() {
				So(tfResourceSchema, ShouldContainKey, "string_prop")
				So(tfResourceSchema["string_prop"].Type, ShouldEqual, schema.TypeString)
			})
			Convey("And the resulted tfResourceSchema should contain status property", func() {
				So(tfResourceSchema, ShouldContainKey, statusDefaultPropertyName)
			})
			Convey("And the resulted tfResourceSchema status field be of type map", func() {
				So(tfResourceSchema[statusDefaultPropertyName].Type, ShouldEqual, schema.TypeMap)
			})
			Convey("And the resulted tfResourceSchema status field should contain all the sub-properties including the id property", func() {
				So(tfResourceSchema[statusDefaultPropertyName].Elem.(*schema.Resource).Schema, ShouldContainKey, "id")
				So(tfResourceSchema[statusDefaultPropertyName].Elem.(*schema.Resource).Schema, ShouldContainKey, "name")
			})
		})
	})

	Convey("Given a swagger schema definition that has few properties including an array of string primitives", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:           "listeners",
					Type:           typeList,
					ArrayItemsType: typeString,
				},
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the  err returned should not be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the resulted tfResourceSchema should not contain ID property", func() {
				So(tfResourceSchema, ShouldNotContainKey, "id")
			})
			Convey("And the resulted tfResourceSchema should contain the array property", func() {
				So(tfResourceSchema, ShouldContainKey, "listeners")
			})
			Convey("And the resulted tfResourceSchema listeners field should be of type list and contain the right item schema string type", func() {
				So(tfResourceSchema["listeners"].Type, ShouldEqual, schema.TypeList)
				So(tfResourceSchema["listeners"].Elem.(*schema.Schema).Type, ShouldEqual, schema.TypeString)
			})
		})
	})

	Convey("Given a swagger schema definition that has few properties including an array of objects", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
					Required: true,
				},
				&specSchemaDefinitionProperty{
					Name:           "listeners",
					Type:           typeList,
					ArrayItemsType: typeObject,
					SpecSchemaDefinition: &specSchemaDefinition{
						Properties: specSchemaDefinitionProperties{
							&specSchemaDefinitionProperty{
								Name: "protocol",
								Type: typeString,
							},
						},
					},
				},
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the  err returned should not be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the resulted tfResourceSchema should not contain ID property", func() {
				So(tfResourceSchema, ShouldNotContainKey, "id")
			})
			Convey("And the resulted tfResourceSchema should contain the array property", func() {
				So(tfResourceSchema, ShouldContainKey, "listeners")
			})
			Convey("And the resulted tfResourceSchema should contain a field of type list and the list should have the right object elem schema of type Resource", func() {
				So(tfResourceSchema["listeners"].Type, ShouldEqual, schema.TypeList)
				So(tfResourceSchema["listeners"].Elem.(*schema.Resource).Schema, ShouldContainKey, "protocol")
				So(tfResourceSchema["listeners"].Elem.(*schema.Resource).Schema["protocol"].Type, ShouldEqual, schema.TypeString)
			})
		})
	})

	Convey("Given a swagger schema definition that has a combination of properties supported", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				idProperty,
				stringProperty,
				intProperty,
				numberProperty,
				boolProperty,
				slicePrimitiveProperty,
				computedProperty,
				optionalProperty,
				sensitiveProperty,
				forceNewProperty,
			},
		}
		Convey("When createResourceSchema method is called", func() {
			tfResourceSchema, err := s.createResourceSchema()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the schema for the resource should contain the expected attributes", func() {
				So(tfResourceSchema, ShouldContainKey, stringProperty.Name)
				So(tfResourceSchema, ShouldContainKey, computedProperty.Name)
				So(tfResourceSchema, ShouldContainKey, intProperty.Name)
				So(tfResourceSchema, ShouldContainKey, numberProperty.Name)
				So(tfResourceSchema, ShouldContainKey, boolProperty.Name)
				So(tfResourceSchema, ShouldContainKey, slicePrimitiveProperty.Name)
				So(tfResourceSchema, ShouldContainKey, optionalProperty.Name)
				So(tfResourceSchema, ShouldContainKey, sensitiveProperty.Name)
				So(tfResourceSchema, ShouldContainKey, forceNewProperty.Name)
			})
			Convey("And the schema property types should match the expected configuration", func() {
				So(tfResourceSchema[stringProperty.Name].Type, ShouldEqual, schema.TypeString)
				So(tfResourceSchema[intProperty.Name].Type, ShouldEqual, schema.TypeInt)
				So(tfResourceSchema[numberProperty.Name].Type, ShouldEqual, schema.TypeFloat)
				So(tfResourceSchema[boolProperty.Name].Type, ShouldEqual, schema.TypeBool)
				So(tfResourceSchema[slicePrimitiveProperty.Name].Type, ShouldEqual, schema.TypeList)
			})
			Convey("And the schema property options should match the expected configuration", func() {
				So(tfResourceSchema[computedProperty.Name].Computed, ShouldBeTrue)
				So(tfResourceSchema[optionalProperty.Name].Optional, ShouldBeTrue)
				So(tfResourceSchema[sensitiveProperty.Name].Sensitive, ShouldBeTrue)
				So(tfResourceSchema[forceNewProperty.Name].ForceNew, ShouldBeTrue)
			})
		})
	})
}

func TestGetImmutableProperties(t *testing.T) {
	Convey("Given resource info is configured with schemaDefinition that contains a property 'immutable_property' that is immutable", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:      "id",
					Type:      typeString,
					ReadOnly:  false,
					Immutable: true,
				},
				&specSchemaDefinitionProperty{
					Name:      "immutable_property",
					Type:      typeString,
					ReadOnly:  false,
					Immutable: true,
				},
			},
		}
		Convey("When getImmutableProperties method is called", func() {
			immutableProperties := s.getImmutableProperties()
			Convey("Then the array returned should contain 'immutable_property'", func() {
				So(immutableProperties, ShouldContain, "immutable_property")
			})
			Convey("And the 'id' property should be ignored even if it's marked as immutable (should never happen though, edge case)", func() {
				So(immutableProperties, ShouldNotContain, "id")
			})
		})
	})

	Convey("Given resource info is configured with schemaDefinition that DOES NOT contain immutable properties", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
				},
				&specSchemaDefinitionProperty{
					Name:      "mutable_property",
					Type:      typeString,
					ReadOnly:  false,
					Immutable: false,
				},
			},
		}
		Convey("When getImmutableProperties method is called", func() {
			immutableProperties := s.getImmutableProperties()
			Convey("Then the array returned should be empty", func() {
				So(immutableProperties, ShouldBeEmpty)
			})
		})
	})

}

func TestGetResourceIdentifier(t *testing.T) {
	Convey("Given a specSchemaDefinition containing a field named id", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: false,
				},
			},
		}
		Convey("When getResourceIdentifier method is called", func() {
			id, err := s.getResourceIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the id returned should be the same as the one in the specSchemaDefinition", func() {
				So(id, ShouldEqual, s.Properties[0].Name)
			})
		})
	})

	Convey("Given a specSchemaDefinition that does not contain a field named id but has a property tagged as IsIdentifier", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:         "someOtherID",
					Type:         typeString,
					ReadOnly:     true,
					IsIdentifier: true,
				},
			},
		}
		Convey("When getResourceIdentifier method is called", func() {
			id, err := s.getResourceIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the id returned should be the same as the one in the specSchemaDefinition", func() {
				So(id, ShouldEqual, s.Properties[0].Name)
			})
		})
	})

	Convey("Given a specSchemaDefinition not containing a field named id nor tagged as identifier", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "someOtherField",
					Type:     typeString,
					ReadOnly: false,
				},
			},
		}
		Convey("When getResourceIdentifier method is called", func() {
			_, err := s.getResourceIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldNotBeNil)
			})
			Convey("Then error message should equal", func() {
				So(err.Error(), ShouldEqual, "could not find any identifier property in the resource schema definition")
			})
		})
	})
}

func TestGetStatusId(t *testing.T) {
	Convey("Given a SpecSchemaDefinition that has an status property that is not an object", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     statusDefaultPropertyName,
					Type:     typeString,
					ReadOnly: true,
				},
			},
		}

		Convey("When getStatusIdentifier method is called", func() {
			statuses, err := s.getStatusIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the statuses returned should not be empty'", func() {
				So(statuses, ShouldNotBeEmpty)
			})
			Convey("Then the value returned should contain the name of the property 'statuses'", func() {
				So(statuses[0], ShouldEqual, statusDefaultPropertyName)
			})
		})
	})

	Convey("Given a swagger schema definition that DOES NOT have an 'status' property but has a property configured with IsStatusIdentifier set to TRUE", t, func() {
		expectedStatusProperty := "some-other-property-holding-status"
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:               expectedStatusProperty,
					Type:               typeString,
					ReadOnly:           true,
					IsStatusIdentifier: true,
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			status, err := s.getStatusIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the status returned should not be empty'", func() {
				So(status, ShouldNotBeEmpty)
			})
			Convey("Then the value returned should contain the name of the property 'some-other-property-holding-status'", func() {
				So(status[0], ShouldEqual, expectedStatusProperty)
			})
		})
	})

	Convey("Given a swagger schema definition that HAS BOTH an 'status' property AND ALSO a property configured with 'x-terraform-field-status' set to true", t, func() {
		expectedStatusProperty := "some-other-property-holding-status"
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     statusDefaultPropertyName,
					Type:     typeString,
					ReadOnly: true,
				},
				&specSchemaDefinitionProperty{
					Name:               expectedStatusProperty,
					Type:               typeString,
					ReadOnly:           true,
					IsStatusIdentifier: true,
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			status, err := s.getStatusIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the status returned should not be empty'", func() {
				So(status, ShouldNotBeEmpty)
			})
			Convey("Then the value returned should be 'some-other-property-holding-status' as it takes preference over the default 'status' property", func() {
				So(status[0], ShouldEqual, expectedStatusProperty)
			})
		})
	})

	Convey("Given a swagger schema definition that HAS an status field which is an object containing a status field", t, func() {
		expectedStatusProperty := "actualStatus"
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     "id",
					Type:     typeString,
					ReadOnly: true,
				},
				&specSchemaDefinitionProperty{
					Name:     statusDefaultPropertyName,
					Type:     typeObject,
					ReadOnly: true,
					SpecSchemaDefinition: &specSchemaDefinition{
						Properties: specSchemaDefinitionProperties{
							&specSchemaDefinitionProperty{
								Name:               expectedStatusProperty,
								Type:               typeString,
								ReadOnly:           true,
								IsStatusIdentifier: true,
							},
						},
					},
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			status, err := s.getStatusIdentifier()
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the status returned should not be empty'", func() {
				So(status, ShouldNotBeEmpty)
			})
			Convey("Then the status array returned should contain the different the trace of property names (hierarchy) until the last one which is the one used as status, in this case 'actualStatus' on the last index", func() {
				So(status[0], ShouldEqual, "status")
				So(status[1], ShouldEqual, expectedStatusProperty)
			})
		})
	})

	Convey("Given a swagger schema definition that DOES NOT have an 'status' property but has a property configured with 'x-terraform-field-status' set to FALSE", t, func() {
		expectedStatusProperty := "some-other-property-holding-status"
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:               expectedStatusProperty,
					Type:               typeString,
					ReadOnly:           true,
					IsStatusIdentifier: false,
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			_, err := s.getStatusIdentifier()
			Convey("Then the error returned should not be nil", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a swagger schema definition that NEITHER HAS an 'status' property NOR a property configured with 'x-terraform-field-status' set to true", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:               "prop-that-is-not-status",
					Type:               typeString,
					ReadOnly:           true,
					IsStatusIdentifier: false,
				},
				&specSchemaDefinitionProperty{
					Name:               "prop-that-is-not-status-and-does-not-have-status-metadata-either",
					Type:               typeString,
					ReadOnly:           true,
					IsStatusIdentifier: false,
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			_, err := s.getStatusIdentifier()
			Convey("Then the error returned should NOT be nil", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a swagger schema definition with a status property that is not readonly", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     statusDefaultPropertyName,
					Type:     typeString,
					ReadOnly: false,
				},
			},
		}
		Convey("When getStatusIdentifier method is called", func() {
			_, err := s.getStatusIdentifier()
			Convey("Then the error returned should NOT be nil", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

}

func TestGetStatusIdentifierFor(t *testing.T) {
	Convey("Given a swagger schema definition with a property configured with 'x-terraform-field-status' set to true and it is not readonly", t, func() {
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:               statusDefaultPropertyName,
					Type:               typeString,
					ReadOnly:           false,
					IsStatusIdentifier: true,
				},
			},
		}
		Convey("When getStatusIdentifierFor method is called with a schema definition and forceReadOnly check is disabled (this happens when the method is called recursively)", func() {
			status, err := s.getStatusIdentifierFor(s, true, false)
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the status array returned should contain the status property even though it's not read only...readonly checks are only perform on root level properties", func() {
				So(status[0], ShouldEqual, "status")
			})
		})
	})
}

func TestGetProperty(t *testing.T) {
	Convey("Given a specSchemaDefinition", t, func() {
		existingPropertyName := "existingPropertyName"
		s := &specSchemaDefinition{
			Properties: specSchemaDefinitionProperties{
				&specSchemaDefinitionProperty{
					Name:     existingPropertyName,
					Type:     typeString,
					ReadOnly: false,
				},
			},
		}
		Convey("When getProperty method is called with an existing property name", func() {
			property, err := s.getProperty(existingPropertyName)
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldBeNil)
			})
			Convey("Then the property returned should be the same as the one in the specSchemaDefinition", func() {
				So(property, ShouldEqual, s.Properties[0])
			})
		})
		Convey("When getProperty method is called with a NON existing property name", func() {
			_, err := s.getProperty("nonExistingPropertyName")
			Convey("Then the error returned should be nil", func() {
				So(err, ShouldNotBeNil)
			})
			Convey("Then the property returned should be the same as the one in the specSchemaDefinition", func() {
				So(err.Error(), ShouldEqual, "property with name 'nonExistingPropertyName' not existing in resource schema definition")
			})
		})
	})
}

func TestGetPropertyBasedOnTerraformName(t *testing.T) {
	existingPropertyName := "existingPropertyName"
	s := &specSchemaDefinition{
		Properties: specSchemaDefinitionProperties{
			&specSchemaDefinitionProperty{
				Name:     existingPropertyName,
				Type:     typeString,
				ReadOnly: false,
			},
		},
	}
	_, err := s.getPropertyBasedOnTerraformName("badTerraformPropertyName")
	assert.EqualError(t, err, "property with terraform name 'badTerraformPropertyName' not existing in resource schema definition")

}
