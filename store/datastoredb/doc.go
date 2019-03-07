/*
Package datastoredb provides an implementation of github.com/alexandre-normand/slackscot/store's StringStorer interface
backed by the Google Cloud Datastore.


Requirements for the Google Cloud Datastore integration:
  - A valid project id with datastore mode enabled
  - Google Cloud Credentials (typically in the form of a json file
    with credentials from https://console.cloud.google.com/apis/credentials/serviceaccountkey)

Note: for deployments using credentials rotation, the current solution supports this use-case
with a naive lazy recreation of the client on error. In order for fresh credentials to be
effective when an authentication error happens, the credential client options must reflect
the fresh credentials. One example of this is

 option.WithCredentialsFile(filename)

Since that option points to a filename, the fresh credentials at that file location
would be refreshed on client recreation.

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
package datastoredb
