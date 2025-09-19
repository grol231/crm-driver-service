// Build script for driver-service-integration-tests
// Handles compilation of protobuf files and other build-time tasks

use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Get the OUT_DIR environment variable
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());

    // Only compile protobuf files if they exist and gRPC is enabled
    let proto_dir = PathBuf::from("proto");
    
    if proto_dir.exists() {
        println!("cargo:rerun-if-changed=proto");
        
        // Find all .proto files
        let proto_files: Vec<_> = std::fs::read_dir(&proto_dir)?
            .filter_map(|entry| {
                let entry = entry.ok()?;
                let path = entry.path();
                if path.extension()? == "proto" {
                    Some(path)
                } else {
                    None
                }
            })
            .collect();

        if !proto_files.is_empty() {
            // Compile protobuf files
            tonic_build::configure()
                .build_server(false) // We're only clients in integration tests
                .build_client(true)
                .out_dir(&out_dir)
                .compile(&proto_files, &[&proto_dir])?;

            println!("Compiled {} protobuf files", proto_files.len());
        }
    }

    // Generate build information
    generate_build_info(&out_dir)?;

    // Set up cargo instructions
    println!("cargo:rerun-if-changed=build.rs");
    println!("cargo:rerun-if-changed=Cargo.toml");
    println!("cargo:rerun-if-env-changed=PROFILE");

    Ok(())
}

fn generate_build_info(out_dir: &PathBuf) -> Result<(), Box<dyn std::error::Error>> {
    let build_info_path = out_dir.join("build_info.rs");
    
    let build_time = chrono::Utc::now().to_rfc3339();
    let git_hash = get_git_hash().unwrap_or_else(|| "unknown".to_string());
    let profile = env::var("PROFILE").unwrap_or_else(|_| "debug".to_string());
    let target = env::var("TARGET").unwrap_or_else(|_| "unknown".to_string());
    
    let build_info = format!(
        r#"
// Auto-generated build information
pub const BUILD_TIME: &str = "{}";
pub const GIT_HASH: &str = "{}";
pub const PROFILE: &str = "{}";
pub const TARGET: &str = "{}";
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
pub const NAME: &str = env!("CARGO_PKG_NAME");

pub fn print_build_info() {{
    println!("Build Information:");
    println!("  Name: {{}}", NAME);
    println!("  Version: {{}}", VERSION);
    println!("  Build Time: {{}}", BUILD_TIME);
    println!("  Git Hash: {{}}", GIT_HASH);
    println!("  Profile: {{}}", PROFILE);
    println!("  Target: {{}}", TARGET);
}}
"#,
        build_time, git_hash, profile, target
    );

    std::fs::write(build_info_path, build_info)?;
    
    Ok(())
}

fn get_git_hash() -> Option<String> {
    use std::process::Command;
    
    let output = Command::new("git")
        .args(&["rev-parse", "HEAD"])
        .output()
        .ok()?;
        
    if output.status.success() {
        let hash = String::from_utf8(output.stdout).ok()?;
        Some(hash.trim().to_string())
    } else {
        None
    }
}