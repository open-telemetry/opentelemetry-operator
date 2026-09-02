FROM docker.io/library/python:3.14-slim@sha256:cae66f2ef0ec51a9891263eeee7f987dacf0a9879e8aa9353d5606e0530619a5

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app.py .

EXPOSE 9123

CMD ["python", "app.py"]
