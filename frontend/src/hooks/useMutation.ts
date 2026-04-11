import { useState } from 'react';

export function useMutation<TInput, TOutput>(
  action: (input: TInput) => Promise<TOutput>
): {
  run: (input: TInput) => Promise<TOutput>;
  loading: boolean;
  error: string | null;
} {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (input: TInput): Promise<TOutput> => {
    setLoading(true);
    setError(null);
    try {
      return await action(input);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'mutation failed');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  return { run, loading, error };
}
