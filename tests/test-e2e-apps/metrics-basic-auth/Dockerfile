FROM docker.io/library/python:3.14-slim@sha256:44dd04494ee8f3b538294360e7c4b3acb87c8268e4d0a4828a6500b1eff50061

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app.py .

EXPOSE 9123

CMD ["python", "app.py"]
