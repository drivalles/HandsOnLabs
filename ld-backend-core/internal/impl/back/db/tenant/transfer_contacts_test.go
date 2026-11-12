package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testSaveTransferContact(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a transfer contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		contact := &ldspec.DbTransferContact{Name: "Sales", PhoneNumber: ldutils.Ptr("+15550000001")}
		require.NoError(t, repo.SaveTransferContact(contact))
		assert.NotZero(t, contact.ID)

		contacts, err := repo.FindTransferContacts()
		require.NoError(t, err)
		require.Len(t, contacts, 1)
		assert.Equal(t, contact.ID, contacts[0].ID)
		assert.Equal(t, "+15550000001", *contacts[0].PhoneNumber)
	})

	t.Run("updates the name and number of a transfer contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		contact := &ldspec.DbTransferContact{Name: "Sales", PhoneNumber: ldutils.Ptr("+15550000001")}
		require.NoError(t, repo.SaveTransferContact(contact))

		require.NoError(t, repo.SaveTransferContact(&ldspec.DbTransferContact{
			Model:       gorm.Model{ID: contact.ID},
			Name:        "Support",
			PhoneNumber: ldutils.Ptr("+15550000002"),
		}))

		contacts, err := repo.FindTransferContacts()
		require.NoError(t, err)
		require.Len(t, contacts, 1)
		assert.Equal(t, "Support", contacts[0].Name)
		assert.Equal(t, "+15550000002", *contacts[0].PhoneNumber)
	})

	t.Run("unknown id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		err := repo.SaveTransferContact(&ldspec.DbTransferContact{Model: gorm.Model{ID: 999}, Name: "Sales"})
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("a conflicting transaction creates the transfer contact again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "transfer_contacts", 1)
		contact := &ldspec.DbTransferContact{Name: "Sales"}

		require.NoError(t, repo.SaveTransferContact(contact))

		assert.EqualValues(t, 2, creates.Load())
		contacts, err := repo.FindTransferContacts()
		require.NoError(t, err)
		require.Len(t, contacts, 1)
		assert.Equal(t, contact.ID, contacts[0].ID)
	})

	t.Run("finds the transfer contacts by name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		for _, name := range []string{"Billing", "Sales", "Support"} {
			require.NoError(t, repo.SaveTransferContact(&ldspec.DbTransferContact{Name: name}))
		}
		contacts, err := repo.FindTransferContacts()
		require.NoError(t, err)
		names := []string{}
		for _, contact := range contacts {
			names = append(names, contact.Name)
		}
		assert.Equal(t, []string{"Billing", "Sales", "Support"}, names)
	})
}
