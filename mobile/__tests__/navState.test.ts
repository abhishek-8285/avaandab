import { deriveNavState } from '../src/utils/navState';
import { Trip } from '../src/types/api';

const makeTrip = (overrides: Partial<Trip> = {}): Trip => ({
  id: 'trip_91',
  tripNumber: 'TRP-8492',
  driverName: 'Rajesh Kumar',
  vehiclePlate: 'MH-12-PQ-4521',
  origin: 'Mumbai Central Depot',
  destination: 'Pune Distribution Hub',
  status: 'PENDING',
  startTime: '2026-08-22T10:30:00Z',
  ...overrides,
});

describe('deriveNavState', () => {
  test('no trip yields an honest empty state (no fabricated data)', () => {
    const nav = deriveNavState(null);
    expect(nav.hasTrip).toBe(false);
    expect(nav.refLabel).toBeNull();
    expect(nav.stepLabel).toBe('—');
    expect(nav.speedKmh).toBeNull();
  });

  test('undefined trip behaves the same as null', () => {
    expect(deriveNavState(undefined).hasTrip).toBe(false);
  });

  test('PENDING trip targets the pickup leg (01/02)', () => {
    const nav = deriveNavState(makeTrip({ status: 'PENDING' }));
    expect(nav.hasTrip).toBe(true);
    expect(nav.legTitle).toBe('HEAD TO PICKUP');
    expect(nav.nextStopAddress).toBe('Mumbai Central Depot');
    expect(nav.stepLabel).toBe('STOP 01/02 · PICKUP');
    expect(nav.refLabel).toBe('REF #TRP-8492');
    expect(nav.statusLine).toBe('TRIP #TRP-8492 · PENDING');
  });

  test('IN_TRANSIT trip targets the drop leg (02/02)', () => {
    const nav = deriveNavState(makeTrip({ status: 'IN_TRANSIT' }));
    expect(nav.legTitle).toBe('DELIVER TO');
    expect(nav.nextStopAddress).toBe('Pune Distribution Hub');
    expect(nav.stepLabel).toBe('STOP 02/02 · DROP');
  });

  test('COMPLETED trip shows delivered state with destination context', () => {
    const nav = deriveNavState(makeTrip({ status: 'COMPLETED' }));
    expect(nav.legTitle).toBe('TRIP DELIVERED');
    expect(nav.stepLabel).toBe('COMPLETE');
    expect(nav.nextStopAddress).toBe('Pune Distribution Hub');
  });

  test('CANCELLED trip shows cancelled state', () => {
    const nav = deriveNavState(makeTrip({ status: 'CANCELLED' }));
    expect(nav.legTitle).toBe('TRIP CANCELLED');
    expect(nav.stepLabel).toBe('COMPLETE');
  });

  test('speed converts m/s to rounded km/h', () => {
    // 8.94 m/s ≈ 32.18 km/h
    expect(deriveNavState(makeTrip(), 8.94).speedKmh).toBe(32);
    expect(deriveNavState(makeTrip(), 0).speedKmh).toBe(0);
  });

  test('negative, NaN and undefined speed stay null (never fabricated)', () => {
    expect(deriveNavState(makeTrip(), -1.2).speedKmh).toBeNull();
    expect(deriveNavState(makeTrip(), NaN).speedKmh).toBeNull();
    expect(deriveNavState(makeTrip(), null).speedKmh).toBeNull();
    expect(deriveNavState(makeTrip()).speedKmh).toBeNull();
  });

  test('missing origin falls back to neutral wording on leg 1', () => {
    const nav = deriveNavState(makeTrip({ status: 'PENDING', origin: '' }));
    expect(nav.nextStopAddress).toBe('Pickup point');
  });

  test('missing destination falls back to neutral wording on leg 2', () => {
    const nav = deriveNavState(makeTrip({ status: 'IN_TRANSIT', destination: '' }));
    expect(nav.nextStopAddress).toBe('Destination');
  });

  test('empty tripNumber omits REF label but keeps status line', () => {
    const nav = deriveNavState(makeTrip({ tripNumber: '' }));
    expect(nav.refLabel).toBeNull();
    expect(nav.statusLine).toBe('STATUS PENDING');
  });
});
