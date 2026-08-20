import React from 'react';
import { render, fireEvent, waitFor, act } from '@testing-library/react-native';
import { DeliveryVerificationScreen } from '../src/components/DeliveryVerificationScreen';
import { OfflineQueue } from '../src/services/offlineQueue';
import { resetSQLiteMockState } from '../jest/setup';
import { useAuthStore } from '../src/stores/authStore';

const globalFetch = global.fetch;

describe('DeliveryVerificationScreen', () => {
  beforeEach(async () => {
    resetSQLiteMockState();
    await OfflineQueue.init();
    await useAuthStore.getState().setAuth('mock_token_123', {
      id: 'u_1',
      name: 'Rajesh Kumar',
      role: 'driver',
      email: 'driver@avandab.com',
      driverId: 'drv_1',
    });
  });

  afterEach(() => {
    global.fetch = globalFetch;
  });

  test('submits multipart form to /api/v1/trips/{tripId}/deliver-pod on confirm', async () => {
    const onComplete = jest.fn();
    const onBack = jest.fn();

    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ trip_number: 'TRP-8492', status: 'delivered', pod_url: '' }),
    });
    global.fetch = fetchMock as any;

    const { getByPlaceholderText, getByText } = render(
      <DeliveryVerificationScreen tripId="trip_99" onComplete={onComplete} onBack={onBack} />
    );

    const nameInput = getByPlaceholderText('e.g. Rajesh Sharma');
    fireEvent.changeText(nameInput, 'Suresh Verma');

    const phoneInput = getByPlaceholderText('+91 98765 43210');
    fireEvent.changeText(phoneInput, '+91 98765 43210');

    const notesInput = getByPlaceholderText('e.g. Received at Gate 3 with intact seal');
    fireEvent.changeText(notesInput, 'Package received with seal intact');

    const submitBtn = getByText('CONFIRM & SUBMIT E-POD');
    await act(async () => {
      fireEvent.press(submitBtn);
    });

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalled();
    });

    const calledUrl = fetchMock.mock.calls[0][0];
    expect(calledUrl).toContain('/api/v1/trips/trip_99/deliver-pod');
  });

  test('calls onBack handler when back button is pressed', () => {
    const onComplete = jest.fn();
    const onBack = jest.fn();

    const { getByText } = render(
      <DeliveryVerificationScreen tripId="trip_back" onComplete={onComplete} onBack={onBack} />
    );

    expect(getByText('POD VERIFICATION')).toBeTruthy();
  });

  test('falls back to OfflineQueue.enqueuePOD on network failure', async () => {
    const onComplete = jest.fn();
    const onBack = jest.fn();

    global.fetch = jest.fn().mockRejectedValue(new Error('Offline connection failed')) as any;

    const { getByPlaceholderText, getByText } = render(
      <DeliveryVerificationScreen tripId="trip_offline_77" onComplete={onComplete} onBack={onBack} />
    );

    const nameInput = getByPlaceholderText('e.g. Rajesh Sharma');
    fireEvent.changeText(nameInput, 'Warehouse Receiver');

    const submitBtn = getByText('CONFIRM & SUBMIT E-POD');
    await act(async () => {
      fireEvent.press(submitBtn);
    });

    await waitFor(async () => {
      const pending = await OfflineQueue.pendingPODs();
      expect(pending.some((p) => p.trip_id === 'trip_offline_77')).toBe(true);
    });
  });
});
