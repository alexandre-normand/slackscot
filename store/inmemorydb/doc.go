/*
Package inmemorydb provides an implementation of github.com/alexandre-normand/slackscot/store's StringStorer interface
as an in-memory data store relying on a wrapping StringStorer for actual persistence.

The main use-case for the inmemorydb is to shield the real StringStorer implementation from receiving too many calls
as plugins may very well query their StringStorer on every message to evaluate for a match or answer. Of course,
using this also allows the slackscot instance to offer lower latency at the expense of increased memory usage.

Most plugin databases are small and this is therefore a good idea to use inmemorydb but if your instance uses
plugins storing a large number of rows, consider using a different storage interface than the slackscot StringStorer
or skipping the usage of the inmemorydb.

Requirements for the Google Cloud Datastore integration:
  - A valid project id with datastore mode enabled
  - Google Cloud Credentials (typically in the form of a json file with credentials from https://console.cloud.google.com/apis/credentials/serviceaccountkey)

Example code:

	import (
		"github.com/alexandre-normand/slackscot/store/datastoredb"
		"github.com/alexandre-normand/slackscot/store/inmemorydb"
		"google.golang.org/api/option"
	)

	func main() {
		// Create your persistent storer first
		persistentStorer, err := datastoredb.New(plugins.KarmaPluginName, "youppi", option.WithCredentialsFile(*gcloudCredentialsFile))
		if err != nil {
			log.Fatalf("Opening [%s] db failed: %s", plugins.KarmaPluginName, err.Error())
		}
		defer persistentStorer.Close()

		// Create the inmemorydb
		karmaStorer, err := inmemorydb.New(persistenStorer)
		if err != nil {
			log.Fatalf("Opening creating in-memory db wrapper: %s", err.Error())
		}

		// Do something with the database
		karma := plugins.NewKarma(karmaStorer)}

		// Run your instance
		...
	}
*/
package inmemorydb
