package nvmf

import (
	"csi-driver-nvmf/pkg/utils"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"os"
)

type NodeServer struct {
	Driver *driver
}

func NewNodeServer(d *driver) *NodeServer {
	return &NodeServer{
		Driver: d,
	}
}

func (n *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.Infof("Using Nvme NodeGetCapabilities")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
					},
				},
			},
		},
	}, nil
}

func (n *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	// 1. check parameters
	targetPath := req.TargetPath
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "NodePublishVolume VolumeID must be provided")
	}

	if targetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "NodePublishVolume Target Path must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Errorf(codes.InvalidArgument, "NodePublishVolume Volume Capability must be provided")
	}

	// 2. attachdisk
	nvmfInfo, err := getNVMfInfo(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodePublishVolume: get NVMf disk info from req err: %v", err)
	}
	diskMounter := getNVMfDiskMounter(nvmfInfo, req)

	// attachDisk realize connect NVMf disk and mount to docker path
	_, err = AttachDisk(req, *diskMounter)
	if err != nil {
		klog.Errorf("NodePublishVolume: Attach volume %s with error: %s", req.VolumeId, err.Error())
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.Infof("NodeUnpublishVolume: Starting unpublish volume, %s, %v", req.VolumeId, req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume VolumeID must be provided")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Staging TargetPath must be provided")
	}
	targetPath := req.GetTargetPath()
	err := DetachDisk(req.VolumeId, getNVMfDiskUnMounter(req), targetPath)
	if err != nil {
		klog.Errorf("NodeUnpublishVolume: VolumeID: %s detachDisk err: %v", req.VolumeId, err)
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	deviceName, err := GetDeviceNameByVolumeID(req.VolumeId)
	if err != nil {
		klog.Errorf("NodeExpandVolume: Get Device by volumeID: %s error %v", req.VolumeId, err)
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume: Get Device by volumeID: %s error %v", req.VolumeId, err)
	}

	scanPath := parseDeviceToControllerPath(deviceName)
	if utils.IsFileExisting(scanPath) {
		file, err := os.OpenFile(scanPath, os.O_RDWR|os.O_TRUNC, 0766)
		err = utils.WriteStringToFile(file, "1")
		if err != nil {
			klog.Errorf("NodeExpandVolume: Rescan error: %v", err)
			return nil, status.Errorf(codes.Internal, "NodeExpandVolume: Rescan error: %v", err)
		}
	} else {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume: rescan path %s not exist", scanPath)
	}

	return &csi.NodeExpandVolumeResponse{}, nil
}

func (n *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: n.Driver.nodeId,
	}, nil
}

func (n *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "NodeGetVolumeStats not implement")
}