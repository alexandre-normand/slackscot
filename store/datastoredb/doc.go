/*
Package databasedb provides an implementation of github.com/alexandre-normand/slackscot/store's StringStorer interface
backed by the Google Cloud Datastore.


Requirements for the Google Cloud Datastore integration:
  - A valid project id with datastore mode enabled
  - Google Cloud Credentials (typically in the form of a json file with credentials from https://console.cloud.google.com/apis/credentials/serviceaccountkey)

Example code:

	import (
		"github.com/alexandre-normand/slackscot/store/datastoredb"
		"google.golang.org/api/option"
	)

	func main() {
		// The first argument is going to be this instance's namespace so the plugin name is a good candidate.
		// The second argument is the gcloud project id which is what you'll have created with your gcloud service account
		// The third argument are client options which are most useful for providing credentials either in the form of a pre-parsed json file or
		// most commonly, the path to a json credentials file
		karmaStorer, err := datastoredb.New(plugins.KarmaPluginName, "youppi", option.WithCredentialsFile(*gcloudCredentialsFile))
		if err != nil {
			log.Fatalf("Opening [%s] db failed: %s", plugins.KarmaPluginName, err.Error())
		}
		defer karmaStorer.Close()

		// Do something with the database
		karma := plugins.NewKarma(karmaStorer)}

		// Run your instance
		...
	}
*/
package datastoredb // import "github.com/alexandre-normand/datastoredb"
