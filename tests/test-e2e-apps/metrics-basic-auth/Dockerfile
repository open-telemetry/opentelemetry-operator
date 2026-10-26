FROM docker.io/library/python:3.15-slim@sha256:533b0b2f9b5d46fd097091cf826f27902f1d6c3bca714b6dc74551356aa97a51

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app.py .

EXPOSE 9123

CMD ["python", "app.py"]
