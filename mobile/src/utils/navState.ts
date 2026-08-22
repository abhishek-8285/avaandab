import { Trip } from '../types/api';

export interface NavState {
  /** True when a real trip backs this screen. */
  hasTrip: boolean;
  /** Current leg label for the instruction card. */
  legTitle: string;
  /** Where the driver is heading right now (real origin/destination name). */
  nextStopAddress: string;
  /** e.g. "STOP 01/02" derived from trip status — never fabricated counts. */
  stepLabel: string;
  /** Real trip reference, e.g. "REF #TRP-8492" (null without trip). */
  refLabel: string | null;
  /** Real GPS speed in km/h rounded, or null when unknown. */
  speedKmh: number | null;
  /** Status line under the brand block. */
  statusLine: string;
}

const mpsToKmh = (mps: number | null | undefined): number | null =>
  typeof mps === 'number' && !isNaN(mps) && mps >= 0 ? Math.round(mps * 3.6) : null;

/**
 * Derives the navigation HUD state from real trip + GPS data only.
 * Every field traces back to backend trip fields or device GPS —
 * the helper never invents distances, ETAs, turn instructions or stop counts.
 */
export function deriveNavState(trip: Trip | null | undefined, speedMps?: number | null): NavState {
  const speedKmh = mpsToKmh(speedMps);

  if (!trip) {
    return {
      hasTrip: false,
      legTitle: 'NO ACTIVE TRIP',
      nextStopAddress: 'Select a trip from the dispatch list',
      stepLabel: '—',
      refLabel: null,
      speedKmh,
      statusLine: 'NO TRIP SELECTED',
    };
  }

  // Two-leg journey: leg 1 = head to pickup (origin), leg 2 = deliver (destination).
  const onLeg1 = trip.status === 'PENDING';
  const finished = trip.status === 'COMPLETED' || trip.status === 'CANCELLED';

  const legTitle = finished
    ? trip.status === 'CANCELLED'
      ? 'TRIP CANCELLED'
      : 'TRIP DELIVERED'
    : onLeg1
      ? 'HEAD TO PICKUP'
      : 'DELIVER TO';

  return {
    hasTrip: true,
    legTitle,
    nextStopAddress: finished ? trip.destination : onLeg1 ? trip.origin || 'Pickup point' : trip.destination || 'Destination',
    stepLabel: finished ? 'COMPLETE' : onLeg1 ? 'STOP 01/02 · PICKUP' : 'STOP 02/02 · DROP',
    refLabel: trip.tripNumber ? `REF #${trip.tripNumber}` : null,
    speedKmh,
    statusLine: trip.tripNumber ? `TRIP #${trip.tripNumber} · ${trip.status}` : `STATUS ${trip.status}`,
  };
}
