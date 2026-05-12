import { useCallback, useEffect, useRef, useState } from 'react'
import { getErrorMessage } from '../utils/error'

interface LoadOptions {
  silent?: boolean
}

interface UseDataLoaderOptions<T> {
  initialData: T
  load: () => Promise<T>
  onError?: (message: string, error: unknown) => void
}

export function useDataLoader<T>({ initialData, load, onError }: UseDataLoaderOptions<T>) {
  const [data, setData] = useState<T>(initialData)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const requestSeqRef = useRef(0)

  const run = useCallback(async (options: LoadOptions = {}) => {
    const { silent = false } = options
    const requestSeq = requestSeqRef.current + 1
    requestSeqRef.current = requestSeq
    const isCurrentRequest = () => requestSeqRef.current === requestSeq

    if (!silent) {
      setLoading(true)
      setError(null)
    }

    try {
      const nextData = await load()
      if (isCurrentRequest()) {
        setData(nextData)
        setError(null)
      }
      return nextData
    } catch (err) {
      const message = getErrorMessage(err)
      if (!silent && isCurrentRequest()) {
        setError(message)
      }
      if (isCurrentRequest()) {
        onError?.(message, err)
      }
      return null
    } finally {
      if (!silent && isCurrentRequest()) {
        setLoading(false)
      }
    }
  }, [load, onError])

  useEffect(() => {
    void run()
  }, [run])

  const reload = useCallback(() => run(), [run])
  const reloadSilently = useCallback(() => run({ silent: true }), [run])

  return {
    data,
    setData,
    loading,
    error,
    reload,
    reloadSilently,
  }
}
