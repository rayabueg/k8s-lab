/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NapPhase enumerates the lifecycle states of a Nap.
// +kubebuilder:validation:Enum=Sleeping;Awake
type NapPhase string

const (
	NapPhaseSleeping NapPhase = "Sleeping"
	NapPhaseAwake    NapPhase = "Awake"
)

// NapSpec defines the desired state of Nap.
type NapSpec struct {
	// Duration is how long the nap lasts. Must parse via Go's time.ParseDuration
	// (e.g. "30s", "5m", "1h30m"). Required.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
	Duration string `json:"duration"`

	// Message is what to log when the nap ends. If empty, a default is used.
	// +optional
	Message string `json:"message,omitempty"`
}

// NapStatus defines the observed state of Nap.
type NapStatus struct {
	// WakesAt is the wall-clock time at which the nap is scheduled to end.
	// +optional
	WakesAt *metav1.Time `json:"wakesAt,omitempty"`

	// Phase is the current lifecycle state.
	// +optional
	Phase NapPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Wakes At",type=string,JSONPath=`.status.wakesAt`
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=`.spec.duration`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Nap is the Schema for the naps API.
type Nap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NapSpec   `json:"spec,omitempty"`
	Status NapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NapList contains a list of Nap.
type NapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nap{}, &NapList{})
}
