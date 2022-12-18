import argparse
import base64
import datetime
import json
import random
import subprocess
import sys
import threading
import time

import requests

parser = argparse.ArgumentParser()
parser.add_argument("-name", help="name of the signal group", type=str)
args = parser.parse_args()

if args.name is None:
    print("name is required")
    sys.exit(1)

url = 'https://tld.iot.hamburg.de/v1.1/Things?%24Filter=name%20eq%20%27' + args.name + '%27&$Expand=Datastreams';
response = requests.get(url)

if response.status_code != 200:
    print("Error: " + str(response.status_code))
    sys.exit(1)

datastreams = {}
for datastream in response.json()['value'][0]['Datastreams']:
    datastreams[f'v1.1/Datastreams({datastream["@iot.id"]})/Observations'] = datastream['properties']['layerName']

sub_udp = f"mosquitto_sub -h tld.iot.hamburg.de -p 1883 -v "
sub_udp += f"-i {str(random.randint(0, 1000000))} "
sub_udp = sub_udp + " ".join([f"-t '{topic}'" for topic in datastreams.keys()])

last_primary_signal_observation = None
last_car_detector_observation = None
last_bike_detector_observation = None

def udp_thread():
    sys.stdout.write(sub_udp + "\n")
    global last_primary_signal_observation
    global last_car_detector_observation
    global last_bike_detector_observation
    process = subprocess.Popen(sub_udp, stdout=subprocess.PIPE, shell=True)
    while True:
        line = process.stdout.readline()
        if not line:
            break
        decoded = line.decode('utf-8').split(" ")
        if datastreams[decoded[0]] == "primary_signal":
            last_primary_signal_observation = json.loads(decoded[1])
        if datastreams[decoded[0]] == "cycle_second":
            sys.stdout.write(" 🔄 New cycle started")
        if datastreams[decoded[0]] == "detector_car":
            last_car_detector_observation = json.loads(decoded[1])
        if datastreams[decoded[0]] == "detector_bike":
            last_car_detector_observation = json.loads(decoded[1])

# Subscribe in parallel to the predictions
sub_predictions = f"mosquitto_sub -h localhost -p 1883 -v "
sub_predictions += f"-i {str(random.randint(0, 1000000))} "
sub_predictions += f"-t 'hamburg/{args.name}' "

last_prediction = None

def predictions_thread():
    sys.stdout.write(sub_predictions + "\n")
    global last_prediction
    process = subprocess.Popen(sub_predictions, stdout=subprocess.PIPE, shell=True)
    while True:
        line = process.stdout.readline()
        if not line:
            break
        decoded = line.decode('utf-8').split(" ")
        last_prediction = json.loads(decoded[1])
        sys.stdout.write(" 🔮 New prediction")

def int_to_color(result): 
    if result == 0:
        return "⚫️"
    if result == 1:
        return "🔴"
    if result == 2:
        return "🟡"
    if result == 3:
        return "🟢"
    if result == 4:
        return "🟠" # Red amber
    if result == 5:
        return "🌟" # Yellow flashing
    if result == 6:
        return "✳️" # Green flashing
    return "{result}"

# Print the state every second
def print_thread():
    global last_primary_signal_observation
    global last_car_detector_observation
    global last_bike_detector_observation
    global last_prediction
    global datastreams
    while True:
        # Format the actual signal state
        actual_str = "⚫️"
        if last_primary_signal_observation:
            result = last_primary_signal_observation["result"]
            actual_str = f"{int_to_color(result)}"
        if "detector_car" in datastreams.values():
            if last_car_detector_observation:
                result = last_car_detector_observation["result"]
                actual_str += f" (🚙 {' ' if result < 100 else ''}{' ' if result < 10 else ''}{result}%)"
            else:
                actual_str += f" (🚙   0%)"
        if "detector_bike" in datastreams.values():
            if last_bike_detector_observation:
                result = last_bike_detector_observation["result"]
                actual_str += f" (🚲 {' ' if result < 100 else ''}{' ' if result < 10 else ''}{result}%)"
            else:
                actual_str += f" (🚲   0%)"

        # Format the prediction
        prediction_str = "Prediction: None"
        if last_prediction is not None:
            now = "".join([int_to_color(c) for c in base64.b64decode(last_prediction["now"])])
            then = "".join([int_to_color(c) for c in base64.b64decode(last_prediction["then"])])
            reference_time = datetime.datetime.strptime(last_prediction["referenceTime"], "%Y-%m-%dT%H:%M:%SZ")
            seconds_passed = (datetime.datetime.now() - reference_time).total_seconds()
            prediction = []
            index = int(seconds_passed)
            if index < len(now):
                i = index % len(now)
                prediction = now[i:] + "🔄" + then
            else:
                i = (index - len(now)) % len(then)
                prediction = then[i:] + "🔄" + then
            prediction = prediction[:60] # Only show the first 60 seconds
            prediction_str = f"{prediction} - P{last_prediction['programId']}"
            
        sys.stdout.write(f"\n{actual_str} ⬅️  {prediction_str}")
        time.sleep(1)

# Start the threads
threads = [
    threading.Thread(target=udp_thread),
    threading.Thread(target=predictions_thread),
    threading.Thread(target=print_thread),
]

for thread in threads:
    thread.start()
for thread in threads:
    thread.join()