# TODO

[*] create secret for .env file and mount it as volume instead of env vars injection at k8s/app.yaml
[*] add footer with privacy policy and terms of service links
[] add webhook for stage.instol.cloud in polar.sh
[*] add title, description and og meta tags to HTML head for better SEO
[] instead of getting polar.sh product id from env var, create producsts table in db with id, name, description, price, polar_stage_product_id, polar_prod_product_id, created_at, updated_at columns and create producsts package to getting product info from db also add seeding sql to the migrations/init.sql file
[*] add loading spinners to deploy instance page at the create instance page.
[] check instace url after deploy because it takes some time to the dns to propagate
[] use golang migrations tool instead of raw sql files
[] make log/slog compatible with the gcp cloud logging log level etc.
[] create cloud build pipeline for automatic deployments on push to main branch
[] keep deployed yaml file name in the instances table to keep track of deployed files
[] improve frontend for better mobile experience
[*] add middlewares for request id, and add log object to the request context
[*] after create instance still pendning status is shown, fix that
[*] sitemap and robots.txt files for better SEO