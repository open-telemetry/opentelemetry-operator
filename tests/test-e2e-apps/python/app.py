from flask import Flask

app = Flask(__name__)

@app.route('/')
def index():
    return "Hi"

if __name__ == "__main__":
    app.run(host='0.0.0.0')
