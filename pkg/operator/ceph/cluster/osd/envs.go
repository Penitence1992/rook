/*
Copyright 2020 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package osd

import (
	"strconv"

	"github.com/pkg/errors"
	opmon "github.com/rook/rook/pkg/operator/ceph/cluster/mon"
	"github.com/rook/rook/pkg/operator/k8sutil"
	v1 "k8s.io/api/core/v1"
)

const (
	osdStoreEnvVarName        = "ROOK_OSD_STORE"
	osdDatabaseSizeEnvVarName = "ROOK_OSD_DATABASE_SIZE"
	osdWalSizeEnvVarName      = "ROOK_OSD_WAL_SIZE"
	osdJournalSizeEnvVarName  = "ROOK_OSD_JOURNAL_SIZE"
	osdsPerDeviceEnvVarName   = "ROOK_OSDS_PER_DEVICE"
	// EncryptedDeviceEnvVarName is used in the pod spec to indicate whether the OSD is encrypted or not
	EncryptedDeviceEnvVarName = "ROOK_ENCRYPTED_DEVICE"
	// CephVolumeEncryptedKeyEnvVarName is the env variable used by ceph-volume to encrypt the OSD (raw mode)
	// Hardcoded in ceph-volume do NOT touch
	CephVolumeEncryptedKeyEnvVarName = "CEPH_VOLUME_DMCRYPT_SECRET"
	osdMetadataDeviceEnvVarName      = "ROOK_METADATA_DEVICE"
	osdWalDeviceEnvVarName           = "ROOK_WAL_DEVICE"
	pvcBackedOSDVarName              = "ROOK_PVC_BACKED_OSD"
	blockPathVarName                 = "ROOK_BLOCK_PATH"
	cvModeVarName                    = "ROOK_CV_MODE"
	lvBackedPVVarName                = "ROOK_LV_BACKED_PV"
	CrushDeviceClassVarName          = "ROOK_OSD_CRUSH_DEVICE_CLASS"
)

func (c *Cluster) getConfigEnvVars(osdProps osdProperties, dataDir string) []v1.EnvVar {
	envVars := []v1.EnvVar{
		nodeNameEnvVar(osdProps.crushHostname),
		{Name: "ROOK_CLUSTER_ID", Value: string(c.clusterInfo.OwnerRef.UID)},
		k8sutil.PodIPEnvVar(k8sutil.PrivateIPEnvVar),
		k8sutil.PodIPEnvVar(k8sutil.PublicIPEnvVar),
		opmon.PodNamespaceEnvVar(c.clusterInfo.Namespace),
		opmon.EndpointEnvVar(),
		opmon.SecretEnvVar(),
		opmon.CephUsernameEnvVar(),
		opmon.CephSecretEnvVar(),
		k8sutil.ConfigDirEnvVar(dataDir),
		k8sutil.ConfigOverrideEnvVar(),
		{Name: "ROOK_FSID", ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: "rook-ceph-mon"},
				Key:                  "fsid",
			},
		}},
		k8sutil.NodeEnvVar(),
	}

	// Give a hint to the prepare pod for what the host in the CRUSH map should be
	crushmapHostname := osdProps.crushHostname
	if !osdProps.portable && osdProps.onPVC() {
		// If it's a pvc that's not portable we only know what the host name should be when inside the osd prepare pod
		crushmapHostname = ""
	}
	envVars = append(envVars, v1.EnvVar{Name: "ROOK_CRUSHMAP_HOSTNAME", Value: crushmapHostname})

	// Append ceph-volume environment variables
	envVars = append(envVars, cephVolumeEnvVar()...)

	if osdProps.storeConfig.DatabaseSizeMB != 0 {
		envVars = append(envVars, v1.EnvVar{Name: osdDatabaseSizeEnvVarName, Value: strconv.Itoa(osdProps.storeConfig.DatabaseSizeMB)})
	}

	if osdProps.storeConfig.WalSizeMB != 0 {
		envVars = append(envVars, v1.EnvVar{Name: osdWalSizeEnvVarName, Value: strconv.Itoa(osdProps.storeConfig.WalSizeMB)})
	}

	if osdProps.storeConfig.OSDsPerDevice != 0 {
		envVars = append(envVars, v1.EnvVar{Name: osdsPerDeviceEnvVarName, Value: strconv.Itoa(osdProps.storeConfig.OSDsPerDevice)})
	}

	if osdProps.storeConfig.EncryptedDevice {
		envVars = append(envVars, v1.EnvVar{Name: EncryptedDeviceEnvVarName, Value: "true"})
	}

	return envVars
}

func (c *Cluster) getDriveGroupEnvVar(osdProps osdProperties) (v1.EnvVar, error) {
	if len(osdProps.driveGroups) == 0 {
		return v1.EnvVar{}, nil
	}

	b, err := MarshalAsDriveGroupBlobs(osdProps.driveGroups)
	if err != nil {
		return v1.EnvVar{}, errors.Wrap(err, "failed to marshal drive groups into an env var")
	}
	return driveGroupsEnvVar(b), nil
}

func nodeNameEnvVar(name string) v1.EnvVar {
	return v1.EnvVar{Name: "ROOK_NODE_NAME", Value: name}
}

func driveGroupsEnvVar(driveGroups string) v1.EnvVar {
	return v1.EnvVar{Name: "ROOK_DRIVE_GROUPS", Value: driveGroups}
}

func dataDevicesEnvVar(dataDevices string) v1.EnvVar {
	return v1.EnvVar{Name: "ROOK_DATA_DEVICES", Value: dataDevices}
}

func deviceFilterEnvVar(filter string) v1.EnvVar {
	return v1.EnvVar{Name: "ROOK_DATA_DEVICE_FILTER", Value: filter}
}

func devicePathFilterEnvVar(filter string) v1.EnvVar {
	return v1.EnvVar{Name: "ROOK_DATA_DEVICE_PATH_FILTER", Value: filter}
}

func metadataDeviceEnvVar(metadataDevice string) v1.EnvVar {
	return v1.EnvVar{Name: osdMetadataDeviceEnvVarName, Value: metadataDevice}
}

func walDeviceEnvVar(walDevice string) v1.EnvVar {
	return v1.EnvVar{Name: osdWalDeviceEnvVarName, Value: walDevice}
}

func pvcBackedOSDEnvVar(pvcBacked string) v1.EnvVar {
	return v1.EnvVar{Name: pvcBackedOSDVarName, Value: pvcBacked}
}

func setDebugLogLevelEnvVar(debug bool) v1.EnvVar {
	level := "INFO"
	if debug {
		level = "DEBUG"
	}
	return v1.EnvVar{Name: "ROOK_LOG_LEVEL", Value: level}
}

func blockPathEnvVariable(lvPath string) v1.EnvVar {
	return v1.EnvVar{Name: blockPathVarName, Value: lvPath}
}

func cvModeEnvVariable(cvMode string) v1.EnvVar {
	return v1.EnvVar{Name: cvModeVarName, Value: cvMode}
}

func lvBackedPVEnvVar(lvBackedPV string) v1.EnvVar {
	return v1.EnvVar{Name: lvBackedPVVarName, Value: lvBackedPV}
}

func crushDeviceClassEnvVar(crushDeviceClass string) v1.EnvVar {
	return v1.EnvVar{Name: CrushDeviceClassVarName, Value: crushDeviceClass}
}

func encryptedDeviceEnvVar(encryptedDevice bool) v1.EnvVar {
	return v1.EnvVar{Name: EncryptedDeviceEnvVarName, Value: strconv.FormatBool(encryptedDevice)}
}

func cephVolumeRawEncryptedEnvVar(pvcName string) v1.EnvVar {
	return v1.EnvVar{
		Name: CephVolumeEncryptedKeyEnvVarName,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{LocalObjectReference: v1.LocalObjectReference{
				Name: generateOSDEncryptionSecretName(pvcName)},
				Key: OsdEncryptionSecretNameKeyName},
		},
	}
}

func cephVolumeEnvVar() []v1.EnvVar {
	return []v1.EnvVar{
		{Name: "CEPH_VOLUME_DEBUG", Value: "1"},
		{Name: "CEPH_VOLUME_SKIP_RESTORECON", Value: "1"},
		// LVM will avoid interaction with udev.
		// LVM will manage the relevant nodes in /dev directly.
		{Name: "DM_DISABLE_UDEV", Value: "1"},
	}
}

func osdActivateEnvVar() []v1.EnvVar {
	monEnvVars := []v1.EnvVar{
		{Name: "ROOK_CEPH_MON_HOST",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{LocalObjectReference: v1.LocalObjectReference{
					Name: "rook-ceph-config"},
					Key: "mon_host"}}},
		{Name: "CEPH_ARGS", Value: "-m $(ROOK_CEPH_MON_HOST)"},
	}

	return append(cephVolumeEnvVar(), monEnvVars...)
}