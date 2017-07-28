/*
Copyright 2017 The Kubernetes Authors.

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

package server

import (
	"github.com/golang/glog"
	"golang.org/x/net/context"

	"k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"

	containerstore "github.com/kubernetes-incubator/cri-containerd/pkg/store/container"
)

// ListContainers lists all containers matching the filter.
func (c *criContainerdService) ListContainers(ctx context.Context, r *runtime.ListContainersRequest) (retRes *runtime.ListContainersResponse, retErr error) {
	glog.V(4).Infof("ListContainers with filter %+v", r.GetFilter())
	defer func() {
		if retErr == nil {
			glog.V(4).Infof("ListContainers returns containers %+v", retRes.GetContainers())
		}
	}()

	// List all containers from store.
	containersInStore := c.containerStore.List()

	var containers []*runtime.Container
	for _, container := range containersInStore {
		containers = append(containers, toCRIContainer(container))
	}

	containers = c.filterCRIContainers(containers, r.GetFilter())
	return &runtime.ListContainersResponse{Containers: containers}, nil
}

// toCRIContainer converts internal container object into CRI container.
func toCRIContainer(container containerstore.Container) *runtime.Container {
	status := container.Status.Get()
	return &runtime.Container{
		Id:           container.ID,
		PodSandboxId: container.SandboxID,
		Metadata:     container.Config.GetMetadata(),
		Image:        container.Config.GetImage(),
		ImageRef:     container.ImageRef,
		State:        status.State(),
		CreatedAt:    status.CreatedAt,
		Labels:       container.Config.GetLabels(),
		Annotations:  container.Config.GetAnnotations(),
	}
}

// filterCRIContainers filters CRIContainers.
func (c *criContainerdService) filterCRIContainers(containers []*runtime.Container, filter *runtime.ContainerFilter) []*runtime.Container {
	if filter == nil {
		return containers
	}

	filtered := []*runtime.Container{}
	for _, cntr := range containers {
		if filter.GetId() != "" && filter.GetId() != cntr.Id {
			continue
		}
		if filter.GetPodSandboxId() != "" && filter.GetPodSandboxId() != cntr.PodSandboxId {
			continue
		}
		if filter.GetState() != nil && filter.GetState().GetState() != cntr.State {
			continue
		}
		if filter.GetLabelSelector() != nil {
			match := true
			for k, v := range filter.GetLabelSelector() {
				got, ok := cntr.Labels[k]
				if !ok || got != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, cntr)
	}

	return filtered
}
