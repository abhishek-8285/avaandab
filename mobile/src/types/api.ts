// Type definitions for Avandab API Data Models

export interface Driver {
  id: string;
  name: string;
  phone: string;
  status: 'available' | 'on_trip' | 'off_duty';
  currentLocation?: {
    latitude: number;
    longitude: number;
  };
}

export interface Trip {
  id: string;
  tripNumber: string;
  driverName: string;
  vehiclePlate: string;
  origin: string;
  destination: string;
  status: 'PENDING' | 'IN_TRANSIT' | 'COMPLETED' | 'CANCELLED';
  startTime: string;
}

export interface Vehicle {
  id: string;
  plateNumber: string;
  model: string;
  capacityKg: number;
  status: 'active' | 'maintenance';
}
