// Copyright 2017 The nats-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"fmt"

	kubernetesutil "github.com/pires/nats-operator/pkg/util/kubernetes"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Cluster) upgradePod(oldPod *v1.Pod) error {
	c.status.AppendUpgradingCondition(c.cluster.Spec.Version, pod.GetName())

	ns := c.cluster.Namespace

	pod, err := c.config.KubeCli.CoreV1().Pods(ns).Get(pod.GetName(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("fail to get pod (%s): %v", pod.GetName(), err)
	}
	oldpod := kubernetesutil.ClonePod(pod)

	c.logger.Infof("upgrading the NATS member %v from %s to %s", pod.GetName(), kubernetesutil.GetNATSVersion(pod), c.cluster.Spec.Version)
	pod.Spec.Containers[0].Image = kubernetesutil.MakeNATSImage(c.cluster.Spec.Version)
	kubernetesutil.SetNATSVersion(pod, c.cluster.Spec.Version)

	patchdata, err := kubernetesutil.CreatePatch(oldpod, pod, v1.Pod{})
	if err != nil {
		return fmt.Errorf("error creating patch: %v", err)
	}

	_, err = c.config.KubeCli.CoreV1().Pods(ns).Patch(pod.GetName(), types.StrategicMergePatchType, patchdata)
	if err != nil {
		return fmt.Errorf("fail to update the NATS member (%s): %v", pod.GetName(), err)
	}
	c.logger.Infof("finished upgrading the NATS member %v", pod.GetName())
	return nil
}
