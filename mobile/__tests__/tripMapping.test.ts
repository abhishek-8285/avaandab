import { mapTripStatus, RawTrip } from '../src/utils/tripMapper';

describe('tripMapper', () => {
  test('maps in_transit to IN_TRANSIT', () => {
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
    const mapped = mapTripStatus(raw);
    expect(mapped.status).toBe('IN_TRANSIT');
    expect(mapped.tripNumber).toBe('TRP-8492');
    expect(mapped.driverName).toBe('Rajesh Kumar');
  });

  test('maps delivered and completed to COMPLETED', () => {
    const rawDelivered: RawTrip = {
      id: 'trip_2',
      trip_number: 'TRP-8493',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'delivered',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(rawDelivered).status).toBe('COMPLETED');

    const rawCompleted: RawTrip = {
      ...rawDelivered,
      status: 'completed',
    };
    expect(mapTripStatus(rawCompleted).status).toBe('COMPLETED');
  });

  test('maps pending and assigned to PENDING', () => {
    const rawPending: RawTrip = {
      id: 'trip_3',
      trip_number: 'TRP-8494',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'pending',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(rawPending).status).toBe('PENDING');

    const rawAssigned: RawTrip = {
      ...rawPending,
      status: 'assigned',
    };
    expect(mapTripStatus(rawAssigned).status).toBe('PENDING');
  });

  test('unknown status defaults to PENDING', () => {
    const raw: RawTrip = {
      id: 'trip_4',
      trip_number: 'TRP-8495',
      driver_name: 'Rajesh Kumar',
      vehicle_plate: 'MH-12-PQ-4521',
      origin: 'Mumbai',
      destination: 'Pune',
      status: 'unknown_custom_state',
      departure_time: '2026-08-20T10:00:00Z',
    };
    expect(mapTripStatus(raw).status).toBe('PENDING');
  });
});
