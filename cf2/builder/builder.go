package main

func main() {
	app.N

}

/*

ln -sf /app /tmp/app
ln -sf /cache /tmp/cache
ln -sf /lifecycle /tmp/lifecycle
ln -sf /dev/null /tmp/output-cache

chown -R vcap:vcap /app /cache

exec /lifecycle/builder "$@"
*/
