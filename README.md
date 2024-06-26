# Predictor

A minimalistic service that syncs signal groups from a SensorThings API and publishes predictions about them. This project is a proof-of-concept for a prediction algorithm that has many advantages over our current "old" prediction service:

- Predictions are published with every possible signal state, not only green and red. This allows for signals that change between other states (e.g. black/red or black/amber) and the displaying of indermediate states (amber-red/amber) in the PrioBike app.
- Predictions are immediately available due to a history persistence which reacts to the currently running program.
- The accuracy of predictions is higher due to inclusion of non-red/green states and a cluster-based algorithm.
- Predictions are published for many more signal groups
- Predictions may react immediately to an occuring signal change that deviates from the initial prediction + they are updated on-demand and not periodically
- The service is highly performant, since it is implemented with Go, and has a low memory-footprint for better scalability

## Quickstart

```
docker-compose up --build
```

## Algorithm

This is a brief introduction to the prediction algorithm. It is separated into the following steps: Synchronization, Observation, Prediction (the actual "algorithm"), and Monitoring.

### 1. Synchronization

The service prefetches the signal groups ("Things") from a SensorThings API, as well as some observations that may have happened before we started our service. An important example is the `signal_program` observation which notes the currently running program. We prefetch this type of observation for every signal group to know which program is currently running. 

### 2. Observation

We connect to the MQTT broker where the Things send their data via MQTT topics ("Datastreams"). We receive the current signal color (`primary_signal`), program (`signal_program`), car/bike detectors (`detector_car`, `detector_bike`) and the end of each cycle (`cycle_second`). When a message arrives on `cycle_second`, we do some error detection/correction and persist the completed data in a vector ("History"). This history serves us as a basis for prediction. The history is also stored according to the currently running program (`signal_program`).

### 3. Prediction

We use a clustering algorithm for signal schedule prediction. A more detailled explanation follows.

For each thing, we continuously check if we need to update our prediction based on the current state of the signal. For example, we must change our prediction quickly if the program changed, by building a new prediction on the specific history for the program. After some time the service should've persisted at least some cycles for each program of every signal. If not, we default to a history where no program was known. 

Based on the best fitting history, we cluster the completed cycles in the history based on their similarity and a distance threshold. Now, we look at the current state of the signal (which color, when in the current cycle?) and find the cluster with the least running distance to our current state. This cluster may consist of many similar cycles, or only one. Then we combine the cycles in the cluster by "collapsing" the cluster. We do this by finding the most prevalent signal color for each second. The collapsed vector is our prediction.

We perform this for the currently running cycle (`predictionNow`), and the cycle after (`predictionThen`). In this way, with a reference start time in the prediction we can predict the signal schedule in the current cycle and at every moment afterwards, by repeating the prediction in `predictionThen`. For example, if `predictionNow` (a vector of colors by second) is 80 seconds long, but the reference time is 180 seconds in the past, we can calculate the index `100 % len(predictionThen)` to extrapolate the predicted state. 

The prediction is published to another MQTT broker, where it can be accessed by every client.

### 4. Monitoring

The prediction is checked against the actual state of the signal and we calculate metrics that are accessible from a monitoring tool (e.g. Prometheus/Grafana). Additionally, we provide debugging tools:

#### Grafana Dashboard

Available under http://localhost:3000

![Screenshot 2022-12-16 at 12 12 04](https://user-images.githubusercontent.com/27271818/208086158-32fde349-991e-45ea-a894-c8f09b5f5ec7.png)

#### Signal Group Monitor

A tool for comparing the prediction with the actual state and the old prediction algorithm. Accessible under http://localhost/monitor.html.

![Screenshot 2022-12-16 at 12 13 47](https://user-images.githubusercontent.com/27271818/208086442-3828b23d-3baa-46bd-868e-89c9282cd41b.png)

#### Signal Group Analyzer

A tool for comparing the prediction with the history of cycles of each signal group and program. Accessible under http://localhost/analyzer.html.

![Screenshot 2022-12-16 at 12 13 04](https://user-images.githubusercontent.com/27271818/208086301-73316182-982f-4563-a723-5183fc0992bd.png)

#### Monitoring Script

Requires `mosquitto_sub` to be installed. Example:

```
python3 observe.py -name 96_22
```

![Screenshot 2022-12-15 at 10 41 58](https://user-images.githubusercontent.com/27271818/207826493-a0a1af0e-d047-4308-a031-92865e313489.png)


 
