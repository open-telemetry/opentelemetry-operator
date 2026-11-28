FROM docker.io/library/python:3.15.2-slim@sha256:08a4c22c92b79ec821a7ad49768480039ee5b7accac3955228e75366e30abc8e

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app.py .

EXPOSE 9123

CMD ["python", "app.py"]
