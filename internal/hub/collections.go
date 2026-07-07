package hub

import (
	"github.com/henrygd/beszel/internal/hub/utils"
	"github.com/pocketbase/pocketbase/core"
)

type collectionRules struct {
	list   *string
	view   *string
	create *string
	update *string
	delete *string
}

// setCollectionAuthSettings applies collection auth settings.
func setCollectionAuthSettings(app core.App) error {
	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	// disable email auth if DISABLE_PASSWORD_AUTH env var is set
	disablePasswordAuth, _ := utils.GetEnv("DISABLE_PASSWORD_AUTH")
	usersCollection.PasswordAuth.Enabled = disablePasswordAuth != "true"
	usersCollection.PasswordAuth.IdentityFields = []string{"email"}
	usersCollection.CreateRule = nil

	if err := app.Save(usersCollection); err != nil {
		return err
	}

	authenticatedRule := "@request.auth.id != \"\""
	systemsReadRule := authenticatedRule
	systemsWriteRule := systemsReadRule + " && @request.auth.role != \"readonly\""

	if err := applyCollectionRules(app, []string{"systems"}, collectionRules{
		list:   &systemsReadRule,
		view:   &systemsReadRule,
		create: &systemsWriteRule,
		update: &systemsWriteRule,
		delete: &systemsWriteRule,
	}); err != nil {
		return err
	}

	if err := applyCollectionRules(app, []string{"containers", "container_stats", "system_stats"}, collectionRules{
		list: &systemsReadRule,
	}); err != nil {
		return err
	}

	if err := applyCollectionRules(app, []string{"system_details"}, collectionRules{
		list: &systemsReadRule,
		view: &systemsReadRule,
	}); err != nil {
		return err
	}

	return nil
}

func applyCollectionRules(app core.App, collectionNames []string, rules collectionRules) error {
	for _, collectionName := range collectionNames {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}
		collection.ListRule = rules.list
		collection.ViewRule = rules.view
		collection.CreateRule = rules.create
		collection.UpdateRule = rules.update
		collection.DeleteRule = rules.delete
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
