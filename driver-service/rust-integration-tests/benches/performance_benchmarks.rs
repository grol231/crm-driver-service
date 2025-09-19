// Performance Benchmarks for Driver Service Integration Tests
// Run with: cargo bench

use criterion::{black_box, criterion_group, criterion_main, Criterion, BenchmarkId, Throughput};
use tokio::runtime::Runtime;
use std::time::Duration;

use driver_service_integration_tests::{
    init_test_environment, TestConfig, 
    fixtures::{generate_test_drivers, UpdateLocationRequest},
};

fn bench_api_operations(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();
    
    // Initialize test environment
    let env = rt.block_on(async {
        match init_test_environment().await {
            Ok(env) => {
                env.cleanup().await.ok();
                Some(env)
            }
            Err(_) => None
        }
    });

    if env.is_none() {
        println!("Skipping API benchmarks - test environment not available");
        return;
    }

    let env = env.unwrap();

    let mut group = c.benchmark_group("api_operations");
    group.sample_size(50); // Smaller sample size for integration benchmarks
    group.measurement_time(Duration::from_secs(30));

    // Benchmark driver creation
    group.bench_function("create_driver", |b| {
        b.to_async(&rt).iter(|| async {
            let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
            black_box(env.api_client.create_test_driver(&test_driver).await.ok())
        })
    });

    // Benchmark location updates
    group.bench_function("update_location", |b| {
        b.to_async(&rt).iter_with_setup(
            || {
                // Setup: create a driver for each iteration
                rt.block_on(async {
                    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                    env.api_client.create_test_driver(&test_driver).await.ok()
                }).flatten()
            },
            |driver| async move {
                if let Some(driver) = driver {
                    let location_request = UpdateLocationRequest {
                        latitude: 55.7558,
                        longitude: 37.6176,
                        altitude: Some(150.0),
                        accuracy: Some(5.0),
                        speed: Some(30.0),
                        bearing: Some(45.0),
                        timestamp: Some(chrono::Utc::now().timestamp()),
                    };
                    black_box(env.api_client.update_location(driver.id, &location_request).await.ok())
                } else {
                    None
                }
            }
        )
    });

    group.finish();
}

fn bench_database_operations(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();
    
    let env = rt.block_on(async {
        match init_test_environment().await {
            Ok(env) => {
                env.cleanup().await.ok();
                Some(env)
            }
            Err(_) => None
        }
    });

    if env.is_none() {
        println!("Skipping database benchmarks - test environment not available");
        return;
    }

    let env = env.unwrap();

    let mut group = c.benchmark_group("database_operations");
    group.sample_size(100);

    // Benchmark database driver creation
    group.bench_function("db_create_driver", |b| {
        b.to_async(&rt).iter(|| async {
            let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
            black_box(env.database.create_test_driver(&test_driver).await.ok())
        })
    });

    // Benchmark driver lookup by ID
    group.bench_function("db_get_driver", |b| {
        b.to_async(&rt).iter_with_setup(
            || {
                // Setup: create a driver for each iteration
                rt.block_on(async {
                    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                    env.database.create_test_driver(&test_driver).await.ok();
                    test_driver.id
                })
            },
            |driver_id| async move {
                black_box(env.database.get_driver(driver_id).await.ok())
            }
        )
    });

    group.finish();
}

fn bench_throughput_scenarios(c: &mut Criterion) {
    let rt = Runtime::new().unwrap();
    
    let env = rt.block_on(async {
        match init_test_environment().await {
            Ok(env) => {
                env.cleanup().await.ok();
                Some(env)
            }
            Err(_) => None
        }
    });

    if env.is_none() {
        println!("Skipping throughput benchmarks - test environment not available");
        return;
    }

    let env = env.unwrap();

    let mut group = c.benchmark_group("throughput");
    group.sample_size(20);
    group.measurement_time(Duration::from_secs(60));

    // Benchmark concurrent driver creation
    for &size in &[1, 5, 10, 20] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::new("concurrent_driver_creation", size), &size, |b, &size| {
            b.to_async(&rt).iter(|| async {
                let test_drivers = generate_test_drivers(size);
                let mut tasks = Vec::new();

                for driver in test_drivers {
                    let client = env.api_client.clone();
                    let task = tokio::spawn(async move {
                        client.create_test_driver(&driver).await.ok()
                    });
                    tasks.push(task);
                }

                let results = futures::future::join_all(tasks).await;
                black_box(results.len())
            })
        });
    }

    // Benchmark concurrent location updates
    for &size in &[1, 10, 50, 100] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::new("concurrent_location_updates", size), &size, |b, &size| {
            b.to_async(&rt).iter_with_setup(
                || {
                    // Setup: create drivers for location updates
                    rt.block_on(async {
                        let test_drivers = generate_test_drivers(size);
                        let mut created_drivers = Vec::new();
                        
                        for driver in test_drivers {
                            if let Ok(created) = env.api_client.create_test_driver(&driver).await {
                                created_drivers.push(created);
                            }
                        }
                        
                        created_drivers
                    })
                },
                |drivers| async move {
                    let mut tasks = Vec::new();

                    for driver in drivers {
                        let client = env.api_client.clone();
                        let task = tokio::spawn(async move {
                            let location_request = UpdateLocationRequest {
                                latitude: 55.7558 + (rand::random::<f64>() - 0.5) * 0.01,
                                longitude: 37.6176 + (rand::random::<f64>() - 0.5) * 0.01,
                                altitude: Some(150.0),
                                accuracy: Some(5.0),
                                speed: Some(30.0),
                                bearing: Some(45.0),
                                timestamp: Some(chrono::Utc::now().timestamp()),
                            };
                            client.update_location(driver.id, &location_request).await.ok()
                        });
                        tasks.push(task);
                    }

                    let results = futures::future::join_all(tasks).await;
                    black_box(results.len())
                }
            )
        });
    }

    group.finish();
}

fn bench_data_generation(c: &mut Criterion) {
    let mut group = c.benchmark_group("data_generation");
    
    // Benchmark test data generation
    for &size in &[1, 10, 100, 1000] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::new("generate_test_drivers", size), &size, |b, &size| {
            b.iter(|| {
                black_box(generate_test_drivers(size))
            })
        });
    }

    group.finish();
}

fn bench_json_serialization(c: &mut Criterion) {
    let test_drivers = generate_test_drivers(100);
    let location_request = UpdateLocationRequest {
        latitude: 55.7558,
        longitude: 37.6176,
        altitude: Some(150.0),
        accuracy: Some(5.0),
        speed: Some(30.0),
        bearing: Some(45.0),
        timestamp: Some(chrono::Utc::now().timestamp()),
    };

    let mut group = c.benchmark_group("json_operations");

    group.bench_function("serialize_driver", |b| {
        b.iter(|| {
            for driver in &test_drivers {
                black_box(serde_json::to_string(driver).unwrap());
            }
        })
    });

    group.bench_function("serialize_location_request", |b| {
        b.iter(|| {
            black_box(serde_json::to_string(&location_request).unwrap())
        })
    });

    group.finish();
}

criterion_group!(
    benches, 
    bench_api_operations,
    bench_database_operations, 
    bench_throughput_scenarios,
    bench_data_generation,
    bench_json_serialization
);

criterion_main!(benches);