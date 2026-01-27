package xray

import (
	"context"
	"fmt"
	"sync"

	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/proxy"

	"github.com/remnawave/node-go/internal/logger"
)

// UserManager handles adding/removing users from xray-core inbounds.
// It uses the Feature API to interact with xray-core directly.
type UserManager struct {
	mu  sync.RWMutex
	ibm inbound.Manager
	log *logger.Logger
}

// NewUserManager creates a UserManager from an xray-core inbound manager.
func NewUserManager(ibm inbound.Manager, log *logger.Logger) *UserManager {
	return &UserManager{
		ibm: ibm,
		log: log,
	}
}

// getProxyUserManager retrieves the UserManager interface for a specific inbound tag.
// This follows the XrayR pattern:
// 1. Get handler by tag from InboundManager
// 2. Cast to proxy.GetInbound
// 3. Get inbound and cast to proxy.UserManager
func (m *UserManager) getProxyUserManager(ctx context.Context, tag string) (proxy.UserManager, error) {
	handler, err := m.ibm.GetHandler(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("no such inbound tag '%s': %w", tag, err)
	}

	// Cast to proxy.GetInbound interface
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return nil, fmt.Errorf("handler '%s' has not implemented proxy.GetInbound", tag)
	}

	// Get the actual inbound and cast to UserManager
	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return nil, fmt.Errorf("handler '%s' has not implemented proxy.UserManager", tag)
	}

	return userManager, nil
}

// AddUser adds a single user to the specified inbound.
// The user must have Account set via serial.ToTypedMessage().
func (m *UserManager) AddUser(ctx context.Context, tag string, user *protocol.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userManager, err := m.getProxyUserManager(ctx, tag)
	if err != nil {
		return err
	}

	// Convert to MemoryUser before adding
	mUser, err := user.ToMemoryUser()
	if err != nil {
		return fmt.Errorf("failed to convert user to memory user: %w", err)
	}

	if err := userManager.AddUser(ctx, mUser); err != nil {
		return fmt.Errorf("failed to add user '%s' to inbound '%s': %w", user.Email, tag, err)
	}

	if m.log != nil {
		m.log.WithField("inbound", tag).WithField("email", user.Email).
			Debug("User added to inbound")
	}

	return nil
}

// AddUsers adds multiple users to the specified inbound.
func (m *UserManager) AddUsers(ctx context.Context, tag string, users []*protocol.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userManager, err := m.getProxyUserManager(ctx, tag)
	if err != nil {
		return err
	}

	for _, user := range users {
		mUser, err := user.ToMemoryUser()
		if err != nil {
			return fmt.Errorf("failed to convert user '%s' to memory user: %w", user.Email, err)
		}

		if err := userManager.AddUser(ctx, mUser); err != nil {
			return fmt.Errorf("failed to add user '%s' to inbound '%s': %w", user.Email, tag, err)
		}
	}

	if m.log != nil {
		m.log.WithField("inbound", tag).WithField("count", len(users)).
			Debug("Users added to inbound")
	}

	return nil
}

// RemoveUser removes a single user from the specified inbound by email.
func (m *UserManager) RemoveUser(ctx context.Context, tag, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userManager, err := m.getProxyUserManager(ctx, tag)
	if err != nil {
		return err
	}

	if err := userManager.RemoveUser(ctx, email); err != nil {
		return fmt.Errorf("failed to remove user '%s' from inbound '%s': %w", email, tag, err)
	}

	if m.log != nil {
		m.log.WithField("inbound", tag).WithField("email", email).
			Debug("User removed from inbound")
	}

	return nil
}

// RemoveUsers removes multiple users from the specified inbound by email.
func (m *UserManager) RemoveUsers(ctx context.Context, tag string, emails []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userManager, err := m.getProxyUserManager(ctx, tag)
	if err != nil {
		return err
	}

	for _, email := range emails {
		if err := userManager.RemoveUser(ctx, email); err != nil {
			// Log but continue - user might already be removed
			if m.log != nil {
				m.log.WithField("inbound", tag).WithField("email", email).
					Warn(fmt.Sprintf("Failed to remove user: %v", err))
			}
		}
	}

	if m.log != nil {
		m.log.WithField("inbound", tag).WithField("count", len(emails)).
			Debug("Users removal completed")
	}

	return nil
}

// RemoveUserFromAllInbounds removes a user from all registered inbound tags.
func (m *UserManager) RemoveUserFromAllInbounds(ctx context.Context, tags []string, email string) error {
	for _, tag := range tags {
		if err := m.RemoveUser(ctx, tag, email); err != nil {
			// Log but continue - user might not exist in this inbound
			if m.log != nil {
				m.log.WithField("inbound", tag).WithField("email", email).
					Debug(fmt.Sprintf("Could not remove user from inbound: %v", err))
			}
		}
	}
	return nil
}
