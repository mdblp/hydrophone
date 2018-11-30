Hydrophone Dev Docs
===

## HTML files templates and Internationalization

Internationalization of emails has been introduced in Hydrophone through the use of static HTML files that contain placeholders for content to be filled at runtime.
This internationalization is based on the audience language. The audience language follows a logic based on Tidepool user language, browser language and English as a final default.

As a matter of fact, the previous logic of having in-code templates (ie in .go files) for emails has been moved to a logic of having templates generated from static files residing on the file system. A potential evolution can be to have files hosted on a S3 bucket (after pitfall described below is solved).

The framework needs a specific folder to be on the filesystem and referenced by the environment variable `TIDEPOOL_HYDROPHONE_SERVICE`. This folder contains the following subfolders:
* html: html template files
* locales: content in various languages. One file per language that name is under format {language ISO2}.yml
* meta: email structure

### Meta files

Each HTML file has its corresponding meta file. One should ensure:
- templateFileName: the name of the html file that this meta is linked to (this is the actual way of linking HTML and meta)
- subject: the name of the key for the email subject that has its corresponding values translated in the locale files
- contentParts: an array of all the keys that can be localized. The keys name the placeholders found in the html files under form {{.keyName}}
- escapeParts: an array of key names that will be escaped during localizations. There will be no tentative to replace these keys by a localized value. It will then not be taken by the translation engine. This key will instead be replaced by information given programmatically. A good example is if you want to include the name of the user in the middle of a localizable text. Note: these keys cannot be changed without a code change.

### Pitfall

Following the previous logic of having all the templates in memory when the service is starting, this first version of emails based on HTML templates has the same pitfall. It needs a service restart to take changes in the HTML files into consideration. This limit in dynamic behaviour is logged as an issue in Github already.

One part of the path to have a more dynamic behaviour is already crossed  with the use of the meta files. These meta files ensure:
- we can add more content to the html file without code change
- we can change the names of the HTML files without code change
The current limit, besides the pitfall of having to restart service when template is amended, is when a new type of template is needed (other than existing "signup", "password forget", ...): this would require code change.

### Possible Enhancements

The current logic is to have all the templates loaded at the initialization of the API service. This makes the dynamic behaviour of using static files very limited. Review this logic to have static files be monitored and reloaded whenever a change appears.

Have the templates files hosted in an external repository (eg AWS S3) for ease of changes for non-technical teams.


### Developing with Source Files

Some default templates files for development are in the `templates/html` folder.

You can serve these files however you like for local development. One simple solution is to, from the terminal in the `templates/html` directory, run python's SimpleHTTPServer like this:

```shell
# Python 2
python -m SimpleHTTPServer 8000
# Python 3
python -m http.server 8000
```

At this point, you should be able to view the email in your browser at, for instance, `http://localhost:8000/signup_confirmation.html`.

We also have an `index.html` file set up with links to all the templates.

## Assets (Images) file locations

All the email assets must be stored in a publicly accessible location. We use Amazon S3 buckets for this.  Assets are stored per environment, so we can have different assets on `dev`, `stg`, `int`, and `prd`

The bucket urls follow this pattern:

`https://s3-us-west-2.amazonaws.com/tidepool-[env]-asset/[type]/[file]`

So the logo image for the dev environment may be found at:

`https://s3-us-west-2.amazonaws.com/tidepool-dev-asset/img/tidepool_logo_light_x2.png`

Currently, only the backend engineering team has access to these buckets, so all image change requests should go through the backend engineering team lead.

During development, you should change the image sources to use files in the local `img` folder. This way, you won't need to ask to have the files uploaded to S3 until you're sure they're ready for QA. This is also helpful, as it keeps a record of intended file changes in version control.

## Testing emails

Testing locally requires that you have a temporary AWS SES credentials provide to you by the backend engineering team lead. These credentials must be kept private, as soon as testing is complete, the engineering team lead mush be informed so as to revoke them.

Extreme care must be taken to not commit this to out public git repo. If that were to happen, for any reason or lenght of time, the backend engineering team lead MUST be notified immediately.

The AWS SES configuration is done in the environment variable `TIDEPOOL_HYDROPHONE_SERVICE`.

### Multiple Email Client Testing

It's important to test the final email rendering in as many email clients as possible.  Emails are notorioulsy fickle, and using a testing service such as Litmus or Email on Acid is recommended before going to production with any markup/styling changes.

We currently haven't settled on which of these 2 services to set up an account with. We've tried both. Email on Acid is about half the price, and suits our needs well enough, so we will likely go that route. Litmus, however, is nicer for it's in-place editing to iron out the many difficult issues in Outlook (or really any of the MS mail clients).

### Recommended Future Improvements

For now, what we're doing is better than in-place editing of the templates for the reasons noted above. There are, however, many ways this process could be improved in the future.

The most notable candidate for improvement is to perform the CSS _inlining_ with a local build tool (perhaps Gulp) to avoid relying on a 3rd party online service, and avoid the manual copy/pasting required.

Another would be to share all of the common markup in HTML templates, and piece them together at build time. Again, Gulp could be used for this, and would be rather quick to implement. There is a good writeup [here](https://bitsofco.de/a-gulp-workflow-for-building-html-email/) on one possible approach using gulp. There is even a [github repo](https://github.com/ireade/gulp-email-workflow/tree/master/src/templates) from this example that is meant as a starting point, so we could basically plug our styles and templates in to it and it should be done at that point.

This process would also take care of all of the other small manual final prepartation steps outlined in our current process above.
