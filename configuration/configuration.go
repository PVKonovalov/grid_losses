package configuration

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const EnvVarPrefix = "FLISR_"

func (s *Configuration) ReadFromEnv() error {
	return readFromEnv(reflect.ValueOf(s).Elem(), "")
}

func (s *Configuration) ListEnv() []string {
	return listEnv(reflect.ValueOf(s).Elem(), "")
}

func readFromEnv(v reflect.Value, parentTag string) error {
	var err error

	for i := 0; i < v.Type().NumField() && err == nil; i++ {
		field := v.Type().Field(i)
		val := v.Field(i)

		switch field.Type.Kind() {
		case reflect.Struct:
			err = readFromEnv(val, field.Tag.Get("yaml"))
		default:
			if field.Tag.Get("env") == "true" {
				ptg := strings.Split(parentTag, ",")
				tg := strings.Split(field.Tag.Get("yaml"), ",")

				envVar := strings.ToUpper(ptg[0] + "_" + tg[0])
				envVal := os.Getenv(envVar)

				if envVal != "" {
					switch val.Kind() {
					case reflect.String:
						val.SetString(envVal)
					case reflect.Bool:
						if strings.ToUpper(envVal) == "TRUE" || envVar == "1" || strings.ToUpper(envVal) == "Y" {
							val.SetBool(true)
						} else {
							val.SetBool(false)
						}
					case reflect.Int:
						var intVar int
						if intVar, err = strconv.Atoi(envVal); err == nil {
							val.SetInt(int64(intVar))
						} else {
							err = fmt.Errorf("%s %v", envVar, err)
						}
					case reflect.Slice:
						if val.Type().String() == "[]string" {
							for _, item := range strings.Split(envVal, ",") {
								if item != "" {
									val.Set(reflect.Append(val, reflect.ValueOf(item)))
								}
							}
						} else {
							err = fmt.Errorf("%s: unsupported slice type '%s'", envVar, val.Type().String())
						}
					default:
						err = fmt.Errorf("%s: unsupported type '%s'", envVar, val.Kind().String())
					}
				}
			}
		}
	}
	return err
}

func listEnv(v reflect.Value, parentTag string) []string {

	var envVarList []string

	for i := 0; i < v.Type().NumField(); i++ {
		field := v.Type().Field(i)
		val := v.Field(i)

		switch field.Type.Kind() {
		case reflect.Struct:
			envVarList = append(envVarList, listEnv(val, field.Tag.Get("yaml"))...)
		default:
			if field.Tag.Get("env") == "true" {
				ptg := strings.Split(parentTag, ",")
				tg := strings.Split(field.Tag.Get("yaml"), ",")

				envVarList = append(envVarList, strings.ToUpper(EnvVarPrefix+ptg[0]+"_"+tg[0]))
			}
		}
	}
	return envVarList
}

// LoadFromFile from yml configuration file
func (s *Configuration) LoadFromFile(configurationFile string) error {
	var f *os.File
	var err error

	if f, err = os.Open(configurationFile); err != nil {
		return err
	}

	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err = decoder.Decode(&s); err != nil {
		return err
	}

	return s.ReadFromEnv()
}
