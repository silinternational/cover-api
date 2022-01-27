# Cover API

## Getting Started
This application is based on the [Buffalo](https://gobuffalo.io) Framework, an opinionated MVC framework that comes 
with many / most of the common features needed in a web application / API.

The only system requirement for running this application is Docker. Once the source is cloned, all you have 
to do to get it running is:

1. Copy `local-example.env` to `local.env` and update values as described in the file. Secrets may be provided by another team member via Signal or other secure communication tool.
1. Add `127.0.0.1 minio` to `/etc/hosts` (or equivalent for your OS)
1. Run `make`

At this point you'll have a running instance of this application available at http://localhost:3000.

### Third-party credentials

#### SAML Identity Provider

Fill in the correct values in the SAML_* variables as appropriate for
your own SAML IDP

### Troubleshooting

#### No response from API server 
 
Error message in browser console:

> Failed to load resource: net::ERR_EMPTY_RESPONSE 

or this error in Insomnia:

> Error: Server returned nothing (no headers, no data)
  
Try disabling TLS by setting DISABLE_TLS to true

#### Login error

On login, session values don't seem to be retained. The login redirect URL contains an error like:

> There was a problem logging into your account

with an error logged with key: `ErrorMissingSessionKey`

Make sure `HOST` as defined in `.env` is the same as `API_HOST` defined in the UI `.env` file.

## API Documentation
The source of this application is annotated with special annotations for [go-swagger](https://goswagger.io) to use
to generate a Swagger specification file as well as render it as HTML. 

To generate the swagger spec `swagger/swagger.json` run `make swagger`.

## Access the Database
A container running Adminer (similar to phpMyAdmin but for Postgres) will be running at port 8000 after you run `make`. 
You can access use Adminer to manage the PostgreSQL database using the following login details:

 - URL: http://localhost:8000
 - System: PostgreSQL
 - Server: db
 - Username: cover
 - Password: cover
 - Database: cover

You can check the box for "permanent login" and it'll save these settings in a cookie for next time. 

## Folder Structure
Much of the folder structure is based on 
Buffalo's conventions, however there are several additional folders for other bits of functionality. 

 - `actions/` - All controllers as well as middleware and Buffalo app setup are here. Controllers should be thin and
    only do what is necessary to accept requests, validate them, and call methods on models or other components.
 
 - `config/` - Buffalo config related stuff, used by the `buffalo` command line application when generating models, 
   migrations, etc. This is not used by the Translation Friends API itself though.
   
 - `domain/` - Business domain level code. We define constants, some types, and many helper functions here. This 
   `domain` package helps avoid circular dependencies so for example both `actions` and `models` packages can include
   `domain` for shared types, constants, etc.
   
 - `grifts/` - "Grifts" are a feature of Buffalo and they serve as commands or tasks that can be run from the command
   line and have access to all the core application components. For example the `grifts/db.go` file is used to seed 
   the database for development. Grifts can be run by running `grift db:seed`, where `db:seed` can be replaced by a 
   different grift command as needed.
   
 - `migrations/` - All database tables, schemas, indexes, foreign keys, etc. are created, altered, and dropped through 
   migrations. Migrations are a `buffalo/pop` ORM feature for scripting database changes and tracking applied changes.
   To create a new migration use the buffalo generator by running `buffalo pop generate fizz {name}` where `{name}` 
   describes the migration, something like `add_project_id_column_to_vms`. If you are creating a whole new table that
   will also have a `model` you can generate them both at the same time by generating the model. See instructions 
   below for more info.
   
 - `models/` - All database models and their tests live here. A model is a struct with attributes that may or may not
   map to database fields. Each struct property should have tags to manage serialization to JSON and DB appropriately.
   For example in `User` the `Password` attribute has the tag `json:"-"` to ensure it does not get serialized to JSON
   in API responses. To generate a new model, you can use the Buffalo generate command. For example: 
   `buffalo pop generate model snapshots id:uuid vm_id:uuid nutanix_id:string disk_gb:int deleted_at:timestamp`. Each
   attribute and type can be specified in this way and it'll generate up/down migrations and the model file. Both the
   migration and model will certainly need modifications after generation, but the generator does help avoid some 
   boilerplate stuff and save time. 
   
 - `templates/` - Primarily for storing email templates for notifications, but it can also hold other kinds of 
   templates that may be needed for rendering responses.
   
 - `tmp/` - Used by Buffalo as part of build process.
   
 - `listeners/` - We use Buffalo's [Events](https://gobuffalo.io/en/docs/events/) feature for pub/sub event
   emitting and handling. This allows decoupling certain things like a user being created and an email being sent to
   them. Since we may want to do several things when a particular event occurs we can register multiple listeners to
   respond when the event is emitted. 
   
## Coding Conventions
See [CONTRIBUTING.md](CONTRIBUTING.md)
