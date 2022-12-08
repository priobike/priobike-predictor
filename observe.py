
# Read the parameters from the command line
# -cycle_second, -primary_signal, -signal_program

import argparse
import sys

parser = argparse.ArgumentParser()
parser.add_argument("-name", help="name of the signal group", type=str)
args = parser.parse_args()

if args.name is None:
    print("name is required")
    sys.exit(1)

import requests

url = 'https://tld.iot.hamburg.de/v1.1/Things?%24Filter=name%20eq%20%27' + args.name + '%27&$Expand=Datastreams';
response = requests.get(url)

if response.status_code != 200:
    print("Error: " + str(response.status_code))
    sys.exit(1)

datastreams = {}
for datastream in response.json()['value'][0]['Datastreams']:
    datastreams[f'v1.1/Datastreams({datastream["@iot.id"]})/Observations'] = datastream['properties']['layerName']

import json
import random
import subprocess

command = f"mosquitto_sub -h tld.iot.hamburg.de -p 1883 -v "
command += f"-i {str(random.randint(0, 1000000))} "
command = command + " ".join([f"-t '{topic}'" for topic in datastreams.keys()])

print(command)

# Execute the command and map the output to the layerName
p = subprocess.Popen(command, stdout=subprocess.PIPE, shell=True)
for line in p.stdout:
    layerName = datastreams[line.decode('utf-8').split(" ")[0]]
    value = json.loads(line.decode('utf-8').split(" ")[1])
    result = value["result"]
    t = value["phenomenonTime"]

    if layerName == "primary_signal":
        if result == 0:
            print(f"{t} ⚫️")
        elif result == 1:
            print(f"{t} 🔴")
        elif result == 2:
            print(f"{t} 🟡")
        elif result == 3:
            print(f"{t} 🟢")
        elif result == 4:
            print(f"{t} 🟡")
        elif result == 5:
            print(f"{t} 🟡 (Flashing)")
        elif result == 6:
            print(f"{t} 🟢 (Flashing)")
        else:
            print(f"{t} ⚫️ (Unknown: {result})")
    
    if layerName == "cycle_second":
        print(f"{t} 🔄 New Cycle")
    
    if layerName == "signal_program":
        print(f"{t} 🔢 New Program: {result}")
        
p.wait()