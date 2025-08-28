package controller

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
)

// EventHandlers handles deployment events
type EventHandlers struct{}

// NewEventHandlers creates a new EventHandlers instance
func NewEventHandlers() *EventHandlers {
	return &EventHandlers{}
}

// HandleAdd handles deployment addition events
func (h *EventHandlers) HandleAdd(deployment *appsv1.Deployment) {
	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println("📦 Deployment Added")
	fmt.Printf("🔤 Name:      %s\n", deployment.Name)
	fmt.Printf("📂 Namespace: %s\n", deployment.Namespace)
	fmt.Printf("🆔 UID:       %s\n", deployment.UID)
	fmt.Printf("🕓 Created:   %s\n", deployment.CreationTimestamp.UTC().Format(time.RFC3339))
	fmt.Printf("📊 Replicas:  %d\n", *deployment.Spec.Replicas)
	fmt.Printf("🏷️  Labels:    %v\n", deployment.Labels)
	fmt.Println("────────────────────────────────────────────────────")
}

// HandleDelete handles deployment deletion events
func (h *EventHandlers) HandleDelete(deployment *appsv1.Deployment) {
	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println("🗑️  Deployment DELETED")
	fmt.Printf("🔤 Name:      %s\n", deployment.Name)
	fmt.Printf("📂 Namespace: %s\n", deployment.Namespace)
	fmt.Printf("🆔 UID:       %s\n", deployment.UID)
	fmt.Printf("🕓 Created:   %s\n", deployment.CreationTimestamp.UTC().Format(time.RFC3339))
	if deployment.DeletionTimestamp != nil {
		fmt.Printf("🕓 Deleted:   %s\n", deployment.DeletionTimestamp.UTC().Format(time.RFC3339))
	}
	fmt.Println("────────────────────────────────────────────────────")
}

// HandleUpdate handles deployment update events
func (h *EventHandlers) HandleUpdate(oldDeployment, newDeployment *appsv1.Deployment) {
	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println("🔄 Deployment UPDATED")
	fmt.Printf("🔤 Name:      %s\n", newDeployment.Name)
	fmt.Printf("📂 Namespace: %s\n", newDeployment.Namespace)
	fmt.Printf("🆔 UID:       %s\n", newDeployment.UID)
	fmt.Printf("🕓 Created:   %s\n", newDeployment.CreationTimestamp.UTC().Format(time.RFC3339))

	// Show what changed
	if *oldDeployment.Spec.Replicas != *newDeployment.Spec.Replicas {
		fmt.Printf("📊 Replicas:  %d → %d\n", *oldDeployment.Spec.Replicas, *newDeployment.Spec.Replicas)
	}

	// Check for image changes
	oldImages := h.extractImages(oldDeployment)
	newImages := h.extractImages(newDeployment)
	if !h.stringSlicesEqual(oldImages, newImages) {
		fmt.Printf("🖼️  Images changed:\n")
		fmt.Printf("   Old: %v\n", oldImages)
		fmt.Printf("   New: %v\n", newImages)
	}

	fmt.Println("────────────────────────────────────────────────────")
}

// Helper methods

// extractImages extracts container images from a deployment
func (h *EventHandlers) extractImages(deployment *appsv1.Deployment) []string {
	var images []string

	// Extract from init containers
	for _, container := range deployment.Spec.Template.Spec.InitContainers {
		images = append(images, container.Image)
	}

	// Extract from regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	return images
}

// stringSlicesEqual checks if two string slices are equal
func (h *EventHandlers) stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
