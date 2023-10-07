package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/PVKonovalov/localcache"
	"grid_losses/configuration"
	"grid_losses/llog"
	"grid_losses/webapi"
	"os"
	"strings"
	"time"
)

const ApiGetTopology = "/api/topology/graph"
const ApiGetEquipment = "/api/equipment"
const ApiTimeoutSec = 60

// Resource Types
const (
	ResourceTypeIsNotDefine      int = 0
	ResourceTypeMeasure          int = 1
	ResourceTypeState            int = 2
	ResourceTypeControl          int = 3
	ResourceTypeProtect          int = 4
	ResourceTypeLink             int = 5
	ResourceTypeChangeSetGroup   int = 6
	ResourceTypeReclosing        int = 7
	ResourceTypeStateLineSegment int = 8
)

type EdgeStruct struct {
	EquipmentType           string `json:"equipment_type,omitempty"`
	EquipmentName           string `json:"equipment_name,omitempty"`
	EquipmentId             int    `json:"equipment_id,omitempty"`
	EquipmentTypeId         int    `json:"equipment_type_id,omitempty"`
	EquipmentVoltageClassId int    `json:"equipment_voltage_class_id,omitempty"`
	Id                      int    `json:"id"`
	StateNormal             int    `json:"state_normal"`
	Terminal1               int    `json:"terminal1"`
	Terminal2               int    `json:"terminal2"`
}

type TopologyStruct struct {
	Edge []EdgeStruct `json:"edge"`
	Node []struct {
		EquipmentId             int    `json:"equipment_id,omitempty"`
		EquipmentName           string `json:"equipment_name,omitempty"`
		EquipmentTypeId         int    `json:"equipment_type_id,omitempty"`
		EquipmentVoltageClassId int    `json:"equipment_voltage_class_id,omitempty"`
		Id                      int    `json:"id"`
	} `json:"node"`
}

type EquipmentStruct struct {
	electricalState       uint32
	groundedFrom          map[int]bool
	energizedFrom         map[int]bool
	EquipmentType         string `json:"equipment_type,omitempty"`
	EquipmentVoltageClass string `json:"equipment_voltage_class"`
	Id                    int    `json:"id"`
	Name                  string `json:"name"`
	TypeId                int    `json:"type_id,omitempty"`
	VoltageClassId        int    `json:"voltage_class_id"`
	Resource              []struct {
		Id          int    `json:"id"`
		Point       string `json:"point"`
		PointId     uint64 `json:"point_id"`
		PointTypeId int    `json:"point_type_id"`
		Type        string `json:"type"`
		TypeId      int    `json:"type_id"`
	} `json:"resource,omitempty"`
}

type ResourceStruct struct {
	equipmentId    int
	resourceTypeId int
}

type ThisService struct {
	config                                configuration.Configuration
	topologyProfile                       *TopologyStruct
	equipmentFromEquipmentId              map[int]EquipmentStruct
	pointNameFromPointId                  map[uint64]string
	resourceStructFromPointId             map[uint64]ResourceStruct
	pointFromEquipmentIdAndResourceTypeId map[int]map[int]uint64
	equipmentIdArrayFromResourceTypeId    map[int][]int
	numberOfCBCheckingLink                int
}

// NewService grid Losses service
func NewService() *ThisService {
	return &ThisService{
		equipmentFromEquipmentId:              make(map[int]EquipmentStruct),
		pointNameFromPointId:                  make(map[uint64]string),
		resourceStructFromPointId:             make(map[uint64]ResourceStruct),
		pointFromEquipmentIdAndResourceTypeId: make(map[int]map[int]uint64),
		equipmentIdArrayFromResourceTypeId:    make(map[int][]int),
	}
}

func ParseTopologyData(data []byte) (*TopologyStruct, error) {
	var topologyStruct TopologyStruct
	err := json.Unmarshal(data, &topologyStruct)
	return &topologyStruct, err
}

func ParseEquipmentData(data []byte) (*[]EquipmentStruct, error) {
	var equipmentStructs []EquipmentStruct
	err := json.Unmarshal(data, &equipmentStructs)
	return &equipmentStructs, err
}

// LoadTopologyProfile Loading topologyProfile from configs.configAPIHostList
func (s *ThisService) LoadTopologyProfile(timeoutSec time.Duration, isLoadFromCache bool, cachePath string) error {
	var topologyData []byte

	cache := localcache.New(cachePath)

	if isLoadFromCache {
		llog.Logger.Infof("Loading topologyProfile from local cache (%s)", cachePath)
		profileData, err := cache.Load()
		if err != nil {
			return err
		}
		s.topologyProfile, err = ParseTopologyData(profileData)
		return err
	}

	resultErr := errors.New("unknown error. Check configuration file")

	var err error

	for _, urlAPIHost := range s.config.ConfigApi.Url {
		urlAPIHost = strings.TrimSpace(urlAPIHost)
		api := webapi.Connection{
			Timeout:         timeoutSec,
			BaseUrl:         urlAPIHost,
			HostVirtualName: s.config.ConfigApi.HostName}

		llog.Logger.Debugf("Logon to %s as %s", api.BaseUrl, s.config.ConfigApi.UserName)
		_, err, _ = api.Logon(s.config.ConfigApi.UserName, s.config.ConfigApi.Password)
		if err != nil {
			llog.Logger.Errorf("Failed to logon: %v", err)
			resultErr = err
			continue
		}

		llog.Logger.Debugf("Getting topology profile ...")
		topologyData, err = api.GetProfile(s.config.GridLosses.ApiPrefix + ApiGetTopology)

		if err != nil {
			llog.Logger.Errorf("Failed to get topology profile: %v", err)
			resultErr = err
			continue
		}

		s.topologyProfile, err = ParseTopologyData(topologyData)

		if err != nil {
			llog.Logger.Errorf("Failed to unmarshal topology profile: %v", err)
			resultErr = err
			continue
		} else {
			resultErr = nil
			break
		}
	}

	if resultErr == nil {
		err = cache.Save(topologyData)

		if err != nil {
			llog.Logger.Errorf("Failed to write to local cache (%s)", cachePath)
			resultErr = err
		}
	} else {
		llog.Logger.Errorf("Failed to load topology profile from API host: %v", resultErr)
		llog.Logger.Infof("Loading from local cache (%s)", cachePath)
		profileData, err := cache.Load()

		if err != nil {
			return err
		}

		s.topologyProfile, err = ParseTopologyData(profileData)
		resultErr = err
	}

	if cache.IsChanged {
		llog.Logger.Infof("Configuration changed from the previous loading")
	}

	return resultErr
}

// LoadEquipmentProfile Loading equipment from config.ConfigAPIHostList
func (s *ThisService) LoadEquipmentProfile(timeoutSec time.Duration, isLoadFromCache bool, cachePath string) error {
	var equipmentData []byte
	var equipments *[]EquipmentStruct

	cache := localcache.New(cachePath)

	if isLoadFromCache {
		llog.Logger.Infof("Loading equipment from local cache (%s)", cachePath)
		profileData, err := cache.Load()
		if err != nil {
			return err
		}
		equipments, err = ParseEquipmentData(profileData)
		if err == nil {
			for _, _equipment := range *equipments {
				s.equipmentFromEquipmentId[_equipment.Id] = _equipment
			}
		}
		return err
	}

	var err error
	resultErr := errors.New("unknown error. Check configuration file")

	for _, urlAPIHost := range s.config.ConfigApi.Url {
		urlAPIHost = strings.TrimSpace(urlAPIHost)
		api := webapi.Connection{
			Timeout:         timeoutSec,
			BaseUrl:         urlAPIHost,
			HostVirtualName: s.config.ConfigApi.HostName}

		llog.Logger.Debugf("Logon to %s as %s", api.BaseUrl, s.config.ConfigApi.UserName)
		_, err, _ = api.Logon(s.config.ConfigApi.UserName, s.config.ConfigApi.Password)
		if err != nil {
			llog.Logger.Errorf("Failed to logon: %v", err)
			resultErr = err
			continue
		}

		llog.Logger.Debugf("Getting equipment profile ...")
		equipmentData, err = api.GetProfile(s.config.GridLosses.ApiPrefix + ApiGetEquipment)

		if err != nil {
			llog.Logger.Errorf("Failed to get equipment: %v", err)
			resultErr = err
			continue
		}

		equipments, err = ParseEquipmentData(equipmentData)

		if err != nil {
			llog.Logger.Errorf("Failed to unmarshal equipment: %v", err)
			resultErr = err
			continue
		} else {
			for _, _equipment := range *equipments {
				s.equipmentFromEquipmentId[_equipment.Id] = _equipment
			}
			resultErr = nil
			break
		}
	}

	if resultErr == nil {
		err := cache.Save(equipmentData)

		if err != nil {
			llog.Logger.Errorf("Failed to write to local cache (%s)", cachePath)
			resultErr = err
		}
	} else {
		llog.Logger.Errorf("Failed to load equipment from API host: %v", resultErr)
		llog.Logger.Infof("Loading from local cache (%s)", cachePath)
		profileData, err := cache.Load()

		if err != nil {
			return err
		}

		equipments, err = ParseEquipmentData(profileData)

		if err == nil {
			for _, _equipment := range *equipments {
				s.equipmentFromEquipmentId[_equipment.Id] = _equipment
			}
		}
		resultErr = err
	}

	if cache.IsChanged {
		llog.Logger.Infof("Configuration changed from the previous loading")
	}

	return resultErr
}

func (s *ThisService) CreateInternalParametersFromProfiles() {
	for _, equipment := range s.equipmentFromEquipmentId {
		for _, resource := range equipment.Resource {
			if resource.TypeId == ResourceTypeProtect ||
				resource.TypeId == ResourceTypeReclosing ||
				resource.TypeId == ResourceTypeState ||
				resource.TypeId == ResourceTypeStateLineSegment ||
				resource.TypeId == ResourceTypeLink {

				s.pointNameFromPointId[resource.PointId] = resource.Point

				s.resourceStructFromPointId[resource.PointId] = ResourceStruct{
					equipmentId:    equipment.Id,
					resourceTypeId: resource.TypeId,
				}

				if _, exists := s.pointFromEquipmentIdAndResourceTypeId[equipment.Id]; !exists {
					s.pointFromEquipmentIdAndResourceTypeId[equipment.Id] = make(map[int]uint64)
				}
				s.pointFromEquipmentIdAndResourceTypeId[equipment.Id][resource.TypeId] = resource.PointId

				if resource.TypeId == ResourceTypeLink {
					s.numberOfCBCheckingLink += 1
				}
			}
			if _, exists := s.equipmentIdArrayFromResourceTypeId[resource.TypeId]; !exists {
				s.equipmentIdArrayFromResourceTypeId[resource.TypeId] = make([]int, 0)
			}
			s.equipmentIdArrayFromResourceTypeId[resource.TypeId] = append(s.equipmentIdArrayFromResourceTypeId[resource.TypeId], equipment.Id)
		}
	}
}

func main() {

	s := NewService()

	var err error

	var pathToConfig string
	var isLoadFromCache bool
	var showEnvVars bool

	flag.StringVar(&pathToConfig, "conf", "grid_losses.yml", "path to yml configuration file")
	flag.BoolVar(&isLoadFromCache, "cache", false, "load profile from the local cache")
	flag.BoolVar(&showEnvVars, "env", false, "show a list of configuration parameters loaded from the environment")
	flag.Parse()

	if showEnvVars {
		fmt.Printf("%+v\n", s.config.ListEnv())
		os.Exit(0)
	}

	if err = s.config.LoadFromFile(pathToConfig); err != nil {
		llog.Logger.Fatalf("Failed to read configuration (%s): %v", pathToConfig, err)
	}

	var logLevel llog.Level

	if logLevel, err = llog.ParseLevel(s.config.GridLosses.LogLevel); err != nil {
		llog.Logger.Warnf("Failed to parse log level (%s): %v", s.config.GridLosses.LogLevel, err)
		llog.Logger.SetLevel(llog.DebugLevel)
	} else {
		llog.Logger.SetLevel(logLevel)
	}

	llog.Logger.Infof("Log level: %s", llog.Logger.GetLevel().UpperString())

	if err = s.LoadTopologyProfile(time.Second*ApiTimeoutSec, isLoadFromCache, "cache/flisr-topology.json"); err != nil {
		llog.Logger.Fatalf("Failed to load topology profile: %v", err)
	}

	if err = s.LoadEquipmentProfile(time.Second*ApiTimeoutSec, isLoadFromCache, "cache/flisr-equipment.json"); err != nil {
		llog.Logger.Fatalf("Failed to load equipment profile: %v", err)
	}

	s.CreateInternalParametersFromProfiles()

}
