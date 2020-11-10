package migration_test

import (
	"reflect"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkt-cash/pktd/pktwallet/walletdb"
	"github.com/pkt-cash/pktd/pktwallet/walletdb/migration"
)

type mockMigrationManager struct {
	currentVersion uint32
	versions       []migration.Version
}

var _ migration.Manager = (*mockMigrationManager)(nil)

func (m *mockMigrationManager) Name() string {
	return "mock"
}

func (m *mockMigrationManager) Namespace() walletdb.ReadWriteBucket {
	return nil
}

func (m *mockMigrationManager) CurrentVersion(_ walletdb.ReadBucket) (uint32, er.R) {
	return m.currentVersion, nil
}

func (m *mockMigrationManager) SetVersion(_ walletdb.ReadWriteBucket, version uint32) er.R {
	m.currentVersion = version
	return nil
}

func (m *mockMigrationManager) Versions() []migration.Version {
	return m.versions
}

// TestGetLatestVersion ensures that we can properly retrieve the latest version
// from a slice of versions.
func TestGetLatestVersion(t *testing.T) {

	tests := []struct {
		versions      []migration.Version
		latestVersion uint32
	}{
		{
			versions:      []migration.Version{},
			latestVersion: 0,
		},
		{
			versions: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
			},
			latestVersion: 1,
		},
		{
			versions: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
			latestVersion: 2,
		},
		{
			versions: []migration.Version{
				{
					Number:    2,
					Migration: nil,
				},
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
			},
			latestVersion: 2,
		},
	}

	for i, test := range tests {
		latestVersion := migration.GetLatestVersion(test.versions)
		if latestVersion != test.latestVersion {
			t.Fatalf("test %d: expected latest version %d, got %d",
				i, test.latestVersion, latestVersion)
		}
	}
}

// TestVersionsToApply ensures that the proper versions that needs to be applied
// are returned given the current version.
func TestVersionsToApply(t *testing.T) {

	tests := []struct {
		currentVersion  uint32
		versions        []migration.Version
		versionsToApply []migration.Version
	}{
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
			},
			versionsToApply: nil,
		},
		{
			currentVersion: 1,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
			},
			versionsToApply: nil,
		},
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
			versionsToApply: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
		},
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    2,
					Migration: nil,
				},
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
			},
			versionsToApply: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
		},
	}

	for i, test := range tests {
		versionsToApply := migration.VersionsToApply(
			test.currentVersion, test.versions,
		)

		if !reflect.DeepEqual(versionsToApply, test.versionsToApply) {
			t.Fatalf("test %d: versions to apply mismatch\n"+
				"expected: %v\ngot: %v", i,
				spew.Sdump(test.versionsToApply),
				spew.Sdump(versionsToApply))
		}
	}
}

// TestUpgradeRevert ensures that we are not able to revert to a previous
// version.
func TestUpgradeRevert(t *testing.T) {
	m := &mockMigrationManager{
		currentVersion: 1,
		versions: []migration.Version{
			{
				Number:    0,
				Migration: nil,
			},
		},
	}

	if err := migration.Upgrade(m); !migration.ErrReversion.Is(err) {
		t.Fatalf("expected Upgrade to fail with ErrReversion, got %v",
			err)
	}
}

// TestUpgradeSameVersion ensures that no upgrades happen if the current version
// matches the latest.
func TestUpgradeSameVersion(t *testing.T) {

	m := &mockMigrationManager{
		currentVersion: 1,
		versions: []migration.Version{
			{
				Number:    0,
				Migration: nil,
			},
			{
				Number: 1,
				Migration: func(walletdb.ReadWriteBucket) er.R {
					return er.New("migration should " +
						"not happen due to already " +
						"being on the latest version")
				},
			},
		},
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}
}

// TestUpgradeNewVersion ensures that we can properly upgrade to a newer version
// if available.
func TestUpgradeNewVersion(t *testing.T) {

	versions := []migration.Version{
		{
			Number:    0,
			Migration: nil,
		},
		{
			Number: 1,
			Migration: func(walletdb.ReadWriteBucket) er.R {
				return nil
			},
		},
	}

	m := &mockMigrationManager{
		currentVersion: 0,
		versions:       versions,
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}

	latestVersion := migration.GetLatestVersion(versions)
	if m.currentVersion != latestVersion {
		t.Fatalf("expected current version to match latest: "+
			"current=%d vs latest=%d", m.currentVersion,
			latestVersion)
	}
}

// TestUpgradeMultipleVersions ensures that we can go through multiple upgrades
// in-order to reach the latest version.
func TestUpgradeMultipleVersions(t *testing.T) {

	previousVersion := uint32(0)
	versions := []migration.Version{
		{
			Number:    previousVersion,
			Migration: nil,
		},
		{
			Number: 1,
			Migration: func(walletdb.ReadWriteBucket) er.R {
				if previousVersion != 0 {
					return er.Errorf("expected previous "+
						"version to be %d, got %d", 0,
						previousVersion)
				}

				previousVersion = 1
				return nil
			},
		},
		{
			Number: 2,
			Migration: func(walletdb.ReadWriteBucket) er.R {
				if previousVersion != 1 {
					return er.Errorf("expected previous "+
						"version to be %d, got %d", 1,
						previousVersion)
				}

				previousVersion = 2
				return nil
			},
		},
	}

	m := &mockMigrationManager{
		currentVersion: 0,
		versions:       versions,
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}

	latestVersion := migration.GetLatestVersion(versions)
	if m.currentVersion != latestVersion {
		t.Fatalf("expected current version to match latest: "+
			"current=%d vs latest=%d", m.currentVersion,
			latestVersion)
	}
}
