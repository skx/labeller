* [labeller](#labeller)
   * [Example Script](#example-script)
   * [Scripting Facilities](#scripting-facilities)
* [Building](#building)
* [Configuration](#configuration)
   * [First Run](#first-run)
* [Feedback?](#feedback)


# labeller

This repository contains a simple tool which will allow you to easily add/remove labels to your Gmail-hosted email based on the messages' subject, sender, or recipient.  This is something somebody coming from a self-hosted email setup might enjoy.  In the past I had an elaborate `procmail` setup to filter messages into different folders, and while you can search gmail pretty freely I'm going to need labels as a crutch for at least a short while.

Internally this project uses [a small scripting engine](https://github.com/skx/evalfilter) to run a script against each message we find.  If it wishes the user-script can perform as many label-modifications as it wishes, based upon the details of the specific message.

While we don't have the full power, or complexity, of procmail it will _probably_ be useful enough to handle all the obvious tasks a user might require.



## Example Script

Once you've handled all the project/authentication setup, as described later in this guide, this is the kind of script you can execute:

```
//
// Assuming we get a message from "bob@example.com" we'll add
// two labels "bob" and "example.com"
//
add( FromPart );
add( FromDomain );

//
// Prove we can do "complex" things too - by adding a label
// conditionally, depending upon the contents of the Subject-header.
//
if ( Subject ~= /attic: backup/ ) {
   add( "backups" );

   //
   // Remove the "UNREAD" label, which will mark the
   // message as having been read.
   //
   // Gmail is weird :)
   //
   remove( "UNREAD" );
}

return false;
```

By default the script executes against all messages which are new/unread, and which don't have any existing labels, although you can change the filter which is used to select messages to allow it to run more broadly.

As you might suspect script has access to a number of fields from the message it is processing:

* `From`
  * This contains the email-address of the message-sender.  (e.g. "`steve@example.com`")
  * Additionally `FromPart` contains the local-part of the address (e.g. "`steve`")
  * Additionally `FromDomain` contains the domain of the address (e.g. "`example.com`".
* `Subject`
  * The subject of the message.
* `To`
  * This contains the email-address of the message-recipient.
  * Additionally `ToPart` contains the local-part of the address.
  * Additionally `ToDomain` contains the domain of the address.

Additional fields can be added, if there is a need for them.

By default the script `${HOME}/.labeller.script` is executed, if you wish you may pass the path of a different script to execute.




## Scripting Facilities

The [small scripting language](https://github.com/skx/evalfilter/) we embed is extensible, at the moment there are only two functions added:

* `add(String)`
  * This will add a label to the message, with the given value.
* `remove(String)`
  * This will remove the given label from the message.

Interestingly if you remove the `UNSEEN` label the message will be marked as having been read by the UI.

It is possible we could expose additional primitives in the future.




# Building

You should be able to install this application using the standard golang approach:

    $ go get github.com/skx/labeller

**TODO**: If you prefer you can [download the latest binary](http://github.com/skx/labeller/releases) release, once it has been built.




# Configuration

Since this application works with your Google Mail there's a fair bit of setup to configure the OAUTH, I've tried to document what I did here but if there are gaps I'll try to help.

It should be said that the authentication I'm using was modeled upon the Google quick-start article here:

* https://developers.google.com/gmail/api/quickstart/go

In brief this is what you want to do:

* 1. Login to the google cloud and create a new project, I named mine `procmail`
  * https://console.cloud.google.com/projectcreate
    * You might need to choose your organization first here and in later steps, it all depends how many accounts you have and who they belong to.
* 2 Now you want to go to the credentials page:
  * https://console.developers.google.com/apis/credentials
  * Create new: "Oauth2" client.  Give it a name that you like.
    * I chose type "other" and left the majority of the fields blank.
      * You _might_ have to create a "Consent Screen" first, I think this only applies to those users who are using gsuite.
* 3. Once you've complete the Oauth2 client-creation you'll find a download icon.  Click it
  * This will save something like `client_secret_blah.....googleusercontent.com.json`

Save the downloaded credentials-file as "`~/.labeller.credentials`" and you can then run the application.




## First Run

The first time you run the script you'll be prompted to open a URL with your browser, and grant the permission to read/modify BUT NOT DELETE your email.

Assuming you wish to proceed you'll get shown a token.  Paste that into your console and you're good.

However there is another step.  The first time you've logged in you'll see an error message:

> Error 403: Access Not Configured.
> Gmail API has not been used in project XXXXX before or it is disabled.
>Enable it by visiting https://console.developers.google.com/apis/api/gmail.googleapis.com/overview?project=XXXXXX then retry.

So do that.  And FINALLY you should be good to go.




# Feedback?

Please do file an issue if you have problems, comments, or feature-requests

See also

https://stackoverflow.com/questions/tagged/gmail-api+go


S
