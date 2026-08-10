import { getGraphQLURL } from '../constants/network';

export interface GraphQLTripResponse {
  data: {
    activeTrips: Array<{
      id: string;
      driver_name: string;
      origin: string;
      destination: string;
      status: string;
      cargo_weight_kg: number;
    }>;
    serverTime: string;
  };
}

class GraphQLClient {
  async fetchActiveTrips(): Promise<GraphQLTripResponse> {
    const endpoint = getGraphQLURL();
    console.log('[GRAPHQL FETCH] Target Endpoint:', endpoint);

    const query = `
      query GetActiveTrips {
        activeTrips {
          id
          driver_name
          origin
          destination
          status
          cargo_weight_kg
        }
        serverTime
      }
    `;

    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query }),
    });

    return await res.json();
  }
}

export const GraphQL = new GraphQLClient();
