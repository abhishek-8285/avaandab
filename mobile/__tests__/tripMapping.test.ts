import { mapTripStatus, RawTrip } from '../src/utils/tripMapper';

describe('tripMapper', () => {
  test('maps in_transit, started, and reached_pickup to IN_TRANSIT', () => {
    const raw: RawTrip = {
      id: 'trip_1',
      trip_number: 'TRP-8492',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'in_transit',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(raw).status).toBe('IN_TRANSIT');
    expect(mapTripStatus({ ...raw, status: 'started' }).status).toBe('IN_TRANSIT');
    expect(mapTripStatus({ ...raw, status: 'reached_pickup' }).status).toBe('IN_TRANSIT');
  });

  test('maps delivered and completed to COMPLETED', () => {
    const raw: RawTrip = {
      id: 'trip_2',
      trip_number: 'TRP-8493',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'delivered',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(raw).status).toBe('COMPLETED');
    expect(mapTripStatus({ ...raw, status: 'completed' }).status).toBe('COMPLETED');
  });

  test('maps pending and assigned to PENDING, and cancelled to CANCELLED', () => {
    const raw: RawTrip = {
      id: 'trip_3',
      trip_number: 'TRP-8494',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'pending',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(raw).status).toBe('PENDING');
    expect(mapTripStatus({ ...raw, status: 'assigned' }).status).toBe('PENDING');
    expect(mapTripStatus({ ...raw, status: 'cancelled' }).status).toBe('CANCELLED');
  });

  test('handles null/undefined fallback fields safely', () => {
    const raw: any = {
      id: 'trip_null',
      status: 'custom_status',
    };
    const mapped = mapTripStatus(raw);
    expect(mapped.id).toBe('trip_null');
    expect(mapped.tripNumber).toBe('');
    expect(mapped.driverName).toBe('');
    expect(mapped.vehiclePlate).toBe('');
    expect(mapped.origin).toBe('');
    expect(mapped.destination).toBe('');
    expect(mapped.status).toBe('PENDING');
    expect(mapped.startTime).toBe('');
  });
});
