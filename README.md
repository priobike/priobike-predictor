# Prediction Service

A minimalistic service that syncs the things and publishes predictions.

```
docker-compose up --build
```

And access the generated static files under `localhost/<static file>`.

## Signal states

- 0: Dark
- 1: Red
- 2: Amber
- 3: Green
- 4: Red-Amber
- 5: Amber-Flashing
- 6: Green-Flashing
