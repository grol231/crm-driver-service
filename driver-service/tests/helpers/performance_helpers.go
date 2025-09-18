//go:build integration

package helpers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/domain/services"
	"driver-service/tests/fixtures"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// PerformanceTestHelper помощник для тестирования производительности
type PerformanceTestHelper struct {
	t               *testing.T
	driverService   services.DriverService
	locationService services.LocationService
}

// NewPerformanceTestHelper создает новый PerformanceTestHelper
func NewPerformanceTestHelper(t *testing.T, driverService services.DriverService, locationService services.LocationService) *PerformanceTestHelper {
	return &PerformanceTestHelper{
		t:               t,
		driverService:   driverService,
		locationService: locationService,
	}
}

// BenchmarkResult результат бенчмарка
type BenchmarkResult struct {
	Operation      string
	TotalTime      time.Duration
	OperationCount int
	AvgTime        time.Duration
	OpsPerSecond   float64
	Errors         int
}

// BenchmarkDriverCreation тестирует производительность создания водителей
func (h *PerformanceTestHelper) BenchmarkDriverCreation(ctx context.Context, count int, concurrency int) *BenchmarkResult {
	h.t.Logf("Benchmarking driver creation: %d drivers with %d concurrent workers", count, concurrency)

	start := time.Now()

	// Канал для задач
	jobs := make(chan int, count)
	results := make(chan error, count)

	// Запускаем воркеры
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for jobID := range jobs {
				driver := fixtures.CreateTestDriver()
				driver.Phone = fmt.Sprintf("+7900123%04d", jobID)
				driver.Email = fmt.Sprintf("driver%d@example.com", jobID)
				driver.LicenseNumber = fmt.Sprintf("TEST%06d", jobID)

				_, err := h.driverService.CreateDriver(ctx, driver)
				results <- err
			}
		}(i)
	}

	// Отправляем задачи
	for i := 0; i < count; i++ {
		jobs <- i
	}
	close(jobs)

	// Ждем завершения всех воркеров
	wg.Wait()
	close(results)

	// Подсчитываем ошибки
	errors := 0
	for err := range results {
		if err != nil {
			errors++
			h.t.Logf("Driver creation error: %v", err)
		}
	}

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(count)
	opsPerSecond := float64(count) / totalTime.Seconds()

	result := &BenchmarkResult{
		Operation:      "Driver Creation",
		TotalTime:      totalTime,
		OperationCount: count,
		AvgTime:        avgTime,
		OpsPerSecond:   opsPerSecond,
		Errors:         errors,
	}

	h.logBenchmarkResult(result)
	return result
}

// BenchmarkLocationUpdates тестирует производительность обновления местоположений
func (h *PerformanceTestHelper) BenchmarkLocationUpdates(ctx context.Context, driverID uuid.UUID, count int, concurrency int) *BenchmarkResult {
	h.t.Logf("Benchmarking location updates: %d updates with %d concurrent workers", count, concurrency)

	start := time.Now()

	// Канал для задач
	jobs := make(chan int, count)
	results := make(chan error, count)

	// Запускаем воркеры
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for jobID := range jobs {
				location := fixtures.CreateTestLocation(driverID)
				location.Latitude += float64(jobID) * 0.0001 // Небольшие изменения координат
				location.Longitude += float64(jobID) * 0.0001
				location.RecordedAt = time.Now().Add(time.Duration(jobID) * time.Second)

				err := h.locationService.UpdateLocation(ctx, location)
				results <- err
			}
		}(i)
	}

	// Отправляем задачи
	for i := 0; i < count; i++ {
		jobs <- i
	}
	close(jobs)

	// Ждем завершения всех воркеров
	wg.Wait()
	close(results)

	// Подсчитываем ошибки
	errors := 0
	for err := range results {
		if err != nil {
			errors++
			h.t.Logf("Location update error: %v", err)
		}
	}

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(count)
	opsPerSecond := float64(count) / totalTime.Seconds()

	result := &BenchmarkResult{
		Operation:      "Location Updates",
		TotalTime:      totalTime,
		OperationCount: count,
		AvgTime:        avgTime,
		OpsPerSecond:   opsPerSecond,
		Errors:         errors,
	}

	h.logBenchmarkResult(result)
	return result
}

// BenchmarkBatchLocationUpdates тестирует производительность пакетных обновлений
func (h *PerformanceTestHelper) BenchmarkBatchLocationUpdates(ctx context.Context, driverID uuid.UUID, batchSize int, batchCount int) *BenchmarkResult {
	h.t.Logf("Benchmarking batch location updates: %d batches of %d locations", batchCount, batchSize)

	start := time.Now()
	errors := 0
	totalOperations := batchCount * batchSize

	for i := 0; i < batchCount; i++ {
		locations := fixtures.CreateTestLocationHistory(driverID, batchSize, 1*time.Second)

		// Сдвигаем времена для каждого батча
		baseTime := time.Now().Add(time.Duration(i*batchSize) * time.Second)
		for j, location := range locations {
			location.RecordedAt = baseTime.Add(time.Duration(j) * time.Second)
			location.CreatedAt = location.RecordedAt
		}

		err := h.locationService.BatchUpdateLocations(ctx, locations)
		if err != nil {
			errors++
			h.t.Logf("Batch update error: %v", err)
		}
	}

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(totalOperations)
	opsPerSecond := float64(totalOperations) / totalTime.Seconds()

	result := &BenchmarkResult{
		Operation:      "Batch Location Updates",
		TotalTime:      totalTime,
		OperationCount: totalOperations,
		AvgTime:        avgTime,
		OpsPerSecond:   opsPerSecond,
		Errors:         errors,
	}

	h.logBenchmarkResult(result)
	return result
}

// BenchmarkNearbyDriversSearch тестирует производительность поиска водителей поблизости
func (h *PerformanceTestHelper) BenchmarkNearbyDriversSearch(ctx context.Context, searchCount int, driversCount int) *BenchmarkResult {
	h.t.Logf("Benchmarking nearby drivers search: %d searches among %d drivers", searchCount, driversCount)

	// Подготавливаем данные - создаем водителей с местоположениями
	driverIDs := make([]uuid.UUID, driversCount)
	for i := 0; i < driversCount; i++ {
		driver := fixtures.CreateTestDriver()
		driver.Phone = fmt.Sprintf("+7900%07d", i)
		driver.Email = fmt.Sprintf("perf_driver%d@example.com", i)
		driver.LicenseNumber = fmt.Sprintf("PERF%06d", i)

		createdDriver, err := h.driverService.CreateDriver(ctx, driver)
		require.NoError(h.t, err)
		driverIDs[i] = createdDriver.ID

		// Добавляем местоположение в случайном месте в радиусе 10км от центра
		location := fixtures.CreateTestLocation(createdDriver.ID)
		location.Latitude = 55.7558 + (float64(i%100)-50)*0.001 // Разброс ±50*0.001 градуса
		location.Longitude = 37.6173 + (float64(i%100)-50)*0.001

		err = h.locationService.UpdateLocation(ctx, location)
		require.NoError(h.t, err)
	}

	// Выполняем бенчмарк поиска
	start := time.Now()
	errors := 0

	for i := 0; i < searchCount; i++ {
		// Случайные координаты поиска в районе Москвы
		searchLat := 55.7558 + (float64(i%20)-10)*0.01
		searchLon := 37.6173 + (float64(i%20)-10)*0.01

		_, err := h.locationService.GetNearbyDrivers(ctx, searchLat, searchLon, 5.0, 20)
		if err != nil {
			errors++
			h.t.Logf("Nearby search error: %v", err)
		}
	}

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(searchCount)
	opsPerSecond := float64(searchCount) / totalTime.Seconds()

	result := &BenchmarkResult{
		Operation:      "Nearby Drivers Search",
		TotalTime:      totalTime,
		OperationCount: searchCount,
		AvgTime:        avgTime,
		OpsPerSecond:   opsPerSecond,
		Errors:         errors,
	}

	h.logBenchmarkResult(result)
	return result
}

// LoadTest выполняет нагрузочное тестирование
func (h *PerformanceTestHelper) LoadTest(ctx context.Context, driverID uuid.UUID, duration time.Duration, concurrency int) *BenchmarkResult {
	h.t.Logf("Load testing for %v with %d concurrent workers", duration, concurrency)

	start := time.Now()
	endTime := start.Add(duration)

	// Счетчики
	var totalOps int64
	var errors int64
	var mu sync.Mutex

	// Запускаем воркеры
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			opCounter := 0
			for time.Now().Before(endTime) {
				// Чередуем операции
				switch opCounter % 3 {
				case 0:
					// Обновление местоположения
					location := fixtures.CreateTestLocation(driverID)
					location.Latitude += float64(opCounter) * 0.0001
					err := h.locationService.UpdateLocation(ctx, location)
					if err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				case 1:
					// Получение текущего местоположения
					_, err := h.locationService.GetCurrentLocation(ctx, driverID)
					if err != nil && err != entities.ErrLocationNotFound {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				case 2:
					// Получение водителя
					_, err := h.driverService.GetDriverByID(ctx, driverID)
					if err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				}

				mu.Lock()
				totalOps++
				mu.Unlock()
				opCounter++

				// Небольшая пауза для имитации реальной нагрузки
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(totalOps)
	opsPerSecond := float64(totalOps) / totalTime.Seconds()

	result := &BenchmarkResult{
		Operation:      "Load Test",
		TotalTime:      totalTime,
		OperationCount: int(totalOps),
		AvgTime:        avgTime,
		OpsPerSecond:   opsPerSecond,
		Errors:         int(errors),
	}

	h.logBenchmarkResult(result)
	return result
}

// MemoryUsageTest тестирует использование памяти
func (h *PerformanceTestHelper) MemoryUsageTest(ctx context.Context, operationCount int) {
	h.t.Logf("Testing memory usage with %d operations", operationCount)

	// Получаем начальную статистику памяти
	var startMemStats, endMemStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMemStats)

	// Выполняем операции
	for i := 0; i < operationCount; i++ {
		driver := fixtures.CreateTestDriver()
		driver.Phone = fmt.Sprintf("+7900%07d", i)
		driver.Email = fmt.Sprintf("mem_test_%d@example.com", i)
		driver.LicenseNumber = fmt.Sprintf("MEM%06d", i)

		createdDriver, err := h.driverService.CreateDriver(ctx, driver)
		if err != nil {
			h.t.Logf("Memory test error: %v", err)
			continue
		}

		// Добавляем несколько местоположений
		for j := 0; j < 5; j++ {
			location := fixtures.CreateTestLocation(createdDriver.ID)
			location.Latitude += float64(j) * 0.001
			err := h.locationService.UpdateLocation(ctx, location)
			if err != nil {
				h.t.Logf("Location update error: %v", err)
			}
		}

		// Принудительная сборка мусора каждые 100 операций
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Финальная сборка мусора и измерение памяти
	runtime.GC()
	runtime.ReadMemStats(&endMemStats)

	// Логируем статистику памяти
	h.t.Logf("Memory usage statistics:")
	h.t.Logf("  Alloc: %d KB -> %d KB (diff: %d KB)",
		startMemStats.Alloc/1024, endMemStats.Alloc/1024, (endMemStats.Alloc-startMemStats.Alloc)/1024)
	h.t.Logf("  TotalAlloc: %d KB -> %d KB (diff: %d KB)",
		startMemStats.TotalAlloc/1024, endMemStats.TotalAlloc/1024, (endMemStats.TotalAlloc-startMemStats.TotalAlloc)/1024)
	h.t.Logf("  Sys: %d KB -> %d KB (diff: %d KB)",
		startMemStats.Sys/1024, endMemStats.Sys/1024, (endMemStats.Sys-startMemStats.Sys)/1024)
	h.t.Logf("  NumGC: %d -> %d (diff: %d)",
		startMemStats.NumGC, endMemStats.NumGC, endMemStats.NumGC-startMemStats.NumGC)
}

// logBenchmarkResult логирует результаты бенчмарка
func (h *PerformanceTestHelper) logBenchmarkResult(result *BenchmarkResult) {
	h.t.Logf("=== %s Benchmark Results ===", result.Operation)
	h.t.Logf("Total operations: %d", result.OperationCount)
	h.t.Logf("Total time: %v", result.TotalTime)
	h.t.Logf("Average time per operation: %v", result.AvgTime)
	h.t.Logf("Operations per second: %.2f", result.OpsPerSecond)
	h.t.Logf("Errors: %d", result.Errors)
	h.t.Logf("Success rate: %.2f%%", float64(result.OperationCount-result.Errors)/float64(result.OperationCount)*100)
}

// AssertPerformanceThresholds проверяет пороги производительности
func (h *PerformanceTestHelper) AssertPerformanceThresholds(result *BenchmarkResult, maxAvgTime time.Duration, minOpsPerSecond float64) {
	if result.AvgTime > maxAvgTime {
		h.t.Errorf("%s: Average time %v exceeds threshold %v", result.Operation, result.AvgTime, maxAvgTime)
	}

	if result.OpsPerSecond < minOpsPerSecond {
		h.t.Errorf("%s: Operations per second %.2f below threshold %.2f", result.Operation, result.OpsPerSecond, minOpsPerSecond)
	}

	errorRate := float64(result.Errors) / float64(result.OperationCount)
	if errorRate > 0.01 { // Максимум 1% ошибок
		h.t.Errorf("%s: Error rate %.2f%% exceeds 1%% threshold", result.Operation, errorRate*100)
	}
}

// StressTest выполняет стресс-тестирование с постепенным увеличением нагрузки
func (h *PerformanceTestHelper) StressTest(ctx context.Context, driverID uuid.UUID) {
	h.t.Log("Running stress test with increasing load")

	// Постепенно увеличиваем нагрузку
	concurrencyLevels := []int{1, 5, 10, 20, 50}
	operationsPerLevel := 100

	for _, concurrency := range concurrencyLevels {
		h.t.Logf("Testing with concurrency level: %d", concurrency)

		result := h.BenchmarkLocationUpdates(ctx, driverID, operationsPerLevel, concurrency)

		// Проверяем, что производительность не деградирует критично
		if result.Errors > operationsPerLevel/10 { // Более 10% ошибок
			h.t.Errorf("High error rate at concurrency %d: %d errors out of %d operations",
				concurrency, result.Errors, operationsPerLevel)
		}

		if result.AvgTime > 1*time.Second { // Средняя операция не должна занимать больше секунды
			h.t.Errorf("High latency at concurrency %d: %v average time", concurrency, result.AvgTime)
		}

		// Пауза между уровнями нагрузки
		time.Sleep(1 * time.Second)
	}
}

// DatabaseConnectionPoolTest тестирует пул соединений с базой данных
func (h *PerformanceTestHelper) DatabaseConnectionPoolTest(ctx context.Context, testDB *TestDB) {
	h.t.Log("Testing database connection pool under load")

	const concurrency = 50
	const operationsPerWorker = 20

	start := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerWorker)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Выполняем простой запрос к БД
				var count int
				err := testDB.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM drivers")
				if err != nil {
					errors <- err
				}

				// Небольшая пауза
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Подсчитываем ошибки
	errorCount := 0
	for err := range errors {
		errorCount++
		h.t.Logf("DB connection error: %v", err)
	}

	totalTime := time.Since(start)
	totalOps := concurrency * operationsPerWorker

	h.t.Logf("Database connection pool test results:")
	h.t.Logf("  Total operations: %d", totalOps)
	h.t.Logf("  Total time: %v", totalTime)
	h.t.Logf("  Errors: %d", errorCount)
	h.t.Logf("  Success rate: %.2f%%", float64(totalOps-errorCount)/float64(totalOps)*100)

	// Проверяем статистику пула соединений
	stats := testDB.DB.GetStats()
	h.t.Logf("Connection pool stats:")
	h.t.Logf("  MaxOpenConnections: %d", stats.MaxOpenConnections)
	h.t.Logf("  OpenConnections: %d", stats.OpenConnections)
	h.t.Logf("  InUse: %d", stats.InUse)
	h.t.Logf("  Idle: %d", stats.Idle)
	h.t.Logf("  WaitCount: %d", stats.WaitCount)
	h.t.Logf("  WaitDuration: %v", stats.WaitDuration)
}
