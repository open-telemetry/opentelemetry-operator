FROM python:alpine3.18

COPY requirements.txt .
RUN pip install -r requirements.txt
COPY app.py .

ENTRYPOINT ["flask", "run", "-p", "8080"]
