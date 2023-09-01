// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use std::collections::VecDeque;
use std::env;
use std::fs;
use std::io;
use std::os::unix;
use std::path::Path;
use std::path::PathBuf;
use std::process;

/// A type of copy operation
#[derive(Debug, PartialEq)]
enum CopyType {
    /// equivalent to cp <source> <dest>
    SingleFile,
    /// equivalent to cp -a <source> <dest>
    Archive,
}

/// Encapsulate a copy operation
struct CopyOperation {
    /// The source path
    source: PathBuf,
    /// The destination path
    destination: PathBuf,
    /// The type of copy being performed
    copy_type: CopyType,
}

/// Parse command line arguments and transform into `CopyOperation`
fn parse_args(args: Vec<&str>) -> io::Result<CopyOperation> {
    if !(args.len() == 3 || args.len() == 4 && args[1].eq("-a")) {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "Invalid parameters. Expected cp [-a] <source> <destination>",
        ));
    }

    if args.len() == 4 {
        return Ok(CopyOperation {
            source: PathBuf::from(args[2]),
            destination: PathBuf::from(args[3]),
            copy_type: CopyType::Archive,
        });
    }

    Ok(CopyOperation {
        source: PathBuf::from(args[1]),
        destination: PathBuf::from(args[2]),
        copy_type: CopyType::SingleFile,
    })
}

/// Execute the copy operation
fn do_copy(operation: CopyOperation) -> io::Result<()> {
    match operation.copy_type {
        CopyType::Archive => copy_archive(&operation.source, &operation.destination)?,
        CopyType::SingleFile => fs::copy(&operation.source, &operation.destination).map(|_| ())?,
    };
    Ok(())
}

/// Execute the recursive type of copy operation
fn copy_archive(source: &Path, dest: &Path) -> io::Result<()> {
    // This will cover the case in which the destination exists
    let sanitized_dest: PathBuf = if dest.exists() {
        dest.to_path_buf()
            .join(source.file_name().ok_or(io::Error::new(
                io::ErrorKind::InvalidInput,
                "Invalid source file",
            ))?)
    } else {
        dest.to_path_buf()
    };

    let mut stack = VecDeque::new();
    stack.push_back((source.to_path_buf(), sanitized_dest));

    while let Some((current_source, current_dest)) = stack.pop_back() {
        if current_source.is_symlink() {
            let target = current_source.read_link()?;
            unix::fs::symlink(target, &current_dest)?;
        } else if current_source.is_dir() {
            fs::create_dir(&current_dest)?;
            for entry in fs::read_dir(current_source)? {
                let next_source = entry?.path();
                let next_dest =
                    current_dest
                        .clone()
                        .join(next_source.file_name().ok_or(io::Error::new(
                            io::ErrorKind::InvalidInput,
                            "Invalid source file",
                        ))?);
                stack.push_back((next_source, next_dest));
            }
        } else if current_source.is_file() {
            fs::copy(current_source, current_dest)?;
        }
    }
    Ok(())
}

fn main() {
    let original_args: Vec<String> = env::args().collect();
    let args = original_args.iter().map(|x| x.as_str()).collect();

    let operation = parse_args(args).unwrap_or_else(|err| {
        eprintln!("Error parsing arguments: {err}");
        process::exit(1);
    });

    do_copy(operation).unwrap_or_else(|err| {
        eprintln!("Error copying files: {err}");
        process::exit(2);
    });
}

#[cfg(test)]
mod tests {
    use std::{
        fs,
        io::Write,
        os::unix,
        path::{Path, PathBuf},
    };

    use crate::{do_copy, parse_args, CopyOperation, CopyType};
    use uuid;

    #[test]
    fn test_parser_archive() {
        // prepare
        let input = vec!["cp", "-a", "foo.txt", "dest.txt"];

        // act
        let result = parse_args(input).unwrap();

        // assert
        assert_eq!(result.source, PathBuf::from("foo.txt"));
        assert_eq!(result.destination, PathBuf::from("dest.txt"));
        assert_eq!(result.copy_type, CopyType::Archive)
    }

    #[test]
    fn test_parser_single() {
        // prepare
        let input: Vec<&str> = vec!["cp", "foo.txt", "dest.txt"];

        // act
        let result = parse_args(input).unwrap();

        // assert
        assert_eq!(result.source, PathBuf::from("foo.txt"));
        assert_eq!(result.destination, PathBuf::from("dest.txt"));
        assert_eq!(result.copy_type, CopyType::SingleFile)
    }

    #[test]
    fn parser_failure() {
        // prepare
        let inputs = vec![
            vec!["cp", "-r", "foo.txt", "bar.txt"],
            vec!["cp", "-a", "param1", "param2", "param3"],
            vec!["cp", "param1", "param2", "param3"],
        ];

        for input in inputs.into_iter() {
            // act
            let result = parse_args(input.clone());

            // assert
            assert!(result.is_err(), "input should fail {:?}", input);
        }
    }

    #[test]
    fn test_copy_single() {
        // prepare
        let tempdir = tempfile::tempdir().unwrap();
        let test_base = tempdir.path().to_path_buf();

        create_file(&test_base, "foo.txt");

        let source = test_base.join("foo.txt");
        let dest = test_base.join("bar.txt");
        let single_copy = CopyOperation {
            copy_type: CopyType::SingleFile,
            source: source.clone(),
            destination: dest.clone(),
        };

        // act
        do_copy(single_copy).unwrap();

        // assert
        assert_same_file(&source, &dest)
    }

    #[test]
    fn single_cannot_copy_directory() {
        // prepare
        let tempdir = tempfile::tempdir().unwrap();
        let test_base = tempdir.path().to_path_buf();

        create_dir(&test_base, "somedir");

        // act
        let single_copy = CopyOperation {
            copy_type: CopyType::SingleFile,
            source: test_base.join("somedir"),
            destination: test_base.join("somewhereelse"),
        };
        let result = do_copy(single_copy);

        // assert
        assert!(result.is_err());
    }

    #[test]
    fn test_copy_archive() {
        // prepare
        let tempdir = tempfile::tempdir().unwrap();
        let test_base = tempdir.path().to_path_buf();
        ["foo", "foo/foo0", "foo/foo1", "foo/bar"]
            .iter()
            .for_each(|x| create_dir(&test_base, x));
        let files = [
            "foo/file1.txt",
            "foo/file2.txt",
            "foo/foo1/file3.txt",
            "foo/bar/file4.txt",
        ];
        files.iter().for_each(|x| create_file(&test_base, x));
        [("foo/symlink1.txt", "./file1.txt")]
            .iter()
            .for_each(|(x, y)| create_symlink(&test_base, x, y));

        // act
        let recursive_copy = CopyOperation {
            copy_type: CopyType::Archive,
            source: test_base.join("foo"),
            destination: test_base.join("bar"),
        };
        do_copy(recursive_copy).unwrap();

        // assert
        files.iter().for_each(|x| {
            assert_same_file(
                &test_base.join(x),
                &test_base.join(x.replace("foo/", "bar/")),
            )
        });
        assert_same_file(
            &test_base.join("foo/symlink1.txt"),
            &test_base.join("bar/symlink1.txt"),
        );

        assert_same_link(
            &test_base.join("foo/symlink1.txt"),
            &test_base.join("bar/symlink1.txt"),
        )
    }

    #[test]
    fn test_copy_archive_destination_exists() {
        // prepare
        let tempdir = tempfile::tempdir().unwrap();
        let test_base = tempdir.path().to_path_buf();
        ["foo", "foo/foo0", "foo/foo1", "foo/bar"]
            .iter()
            .for_each(|x| create_dir(&test_base, x));
        let files = [
            "foo/file1.txt",
            "foo/file2.txt",
            "foo/foo1/file3.txt",
            "foo/bar/file4.txt",
        ];
        files.iter().for_each(|x| create_file(&test_base, x));
        [("foo/symlink1.txt", "./file1.txt")]
            .iter()
            .for_each(|(x, y)| create_symlink(&test_base, x, y));
        create_dir(&test_base, "bar");

        // act
        let recursive_copy = CopyOperation {
            copy_type: CopyType::Archive,
            source: test_base.join("foo"),
            destination: test_base.join("bar"),
        };
        do_copy(recursive_copy).unwrap();

        // assert
        files.iter().for_each(|x| {
            assert_same_file(
                &test_base.join(x),
                &test_base.join(x.replace("foo/", "bar/foo/")),
            )
        });

        assert_same_link(
            &test_base.join("foo/symlink1.txt"),
            &test_base.join("bar/foo/symlink1.txt"),
        )
    }

    // Utility functions used in the tests
    fn create_dir(base: &Path, dir: &str) {
        fs::create_dir_all(base.to_path_buf().join(dir)).unwrap();
    }

    fn create_file(base: &Path, file: &str) {
        let mut file = fs::File::create(base.to_path_buf().join(file)).unwrap();
        file.write_fmt(format_args!("{}", uuid::Uuid::new_v4().to_string()))
            .unwrap();
    }

    fn create_symlink(base: &Path, file: &str, target: &str) {
        unix::fs::symlink(Path::new(target), &base.to_path_buf().join(file)).unwrap();
    }

    fn assert_same_file(source: &Path, dest: &Path) {
        assert!(source.exists());
        assert!(dest.exists());
        assert!(source.is_file());
        assert!(dest.is_file());

        assert_eq!(
            fs::read_to_string(source).unwrap(),
            fs::read_to_string(dest).unwrap()
        );
    }

    fn assert_same_link(source: &Path, dest: &Path) {
        assert!(source.exists());
        assert!(dest.exists());
        assert!(source.is_symlink());
        assert!(dest.is_symlink());

        assert_eq!(fs::read_link(source).unwrap(), fs::read_link(dest).unwrap());
    }
}
