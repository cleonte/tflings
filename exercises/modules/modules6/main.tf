# I AM NOT DONE

# The root module calls app. The fix is not in this file, but in the module that
# calls db: see modules/app/main.tf.
# TODO: read the error of init; it names the file and the line.
module "app" {
  source = "./modules/app"
}
