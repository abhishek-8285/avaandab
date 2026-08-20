import { Trip } from '../types/api';

export interface RawTrip {
  id: string;
  trip_number: string;
  driver_name: string;
  vehicle_plate: string;
  origin: string;
  destination: string;
  status: 'pending' | 'assigned' | 'started' | 'reached_pickup' | 'in_transit' | 'delivered' | 'completed' | 'cancelled' | string;
  departure_time: string;
}

// Map backend snake_case status to mobile union
export function mapTripStatus(t: RawTrip): Trip {
  const m: Record<string, Trip['status']> = {
    pending: 'PENDING',
    assigned: 'PENDING',
    started: 'IN_TRANSIT',
    reached_pickup: 'IN_TRANSIT',
    in_transit: 'IN_TRANSIT',
    delivered: 'COMPLETED',
    completed: 'COMPLETED',
    cancelled: 'CANCELLED',
  };
  return {
    id: t.id,
    tripNumber: t.trip_number || '',
    driverName: t.driver_name || '',
    vehiclePlate: t.vehicle_plate || '',
    origin: t.origin || '',
    destination: t.destination || '',
    status: m[t.status] ?? 'PENDING',
    startTime: t.departure_time || '',
  };
}
