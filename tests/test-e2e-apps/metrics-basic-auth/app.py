import os
os.environ['PROMETHEUS_DISABLE_CREATED_SERIES'] = 'true'

from flask import Flask, Response, request
from prometheus_client import Gauge, generate_latest, REGISTRY, PROCESS_COLLECTOR, PLATFORM_COLLECTOR, GC_COLLECTOR

app = Flask(__name__)

REGISTRY.unregister(PROCESS_COLLECTOR)
REGISTRY.unregister(PLATFORM_COLLECTOR)
REGISTRY.unregister(GC_COLLECTOR)

secure = Gauge('authenticated', 'Client was authenticated')
secure.set(1)

USERNAME = "user"
PASSWORD = "t0p$ecreT"

def check_auth(username, password):
    return username == USERNAME and password == PASSWORD

def authenticate():
    return Response(
    'Could not verify your access level for that URL.\n'
    'You have to login with proper credentials', 401,
    {'WWW-Authenticate': 'Basic realm="Login Required"'})

@app.route('/metrics')
def metrics():
    auth = request.authorization
    if not auth or not check_auth(auth.username, auth.password):
        return authenticate()
    return Response(generate_latest(), mimetype='text/plain')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9123)
