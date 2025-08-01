import os
import json
import sys

# --- CUSTOMIZATION SECTION ---
# 
# 1. Define the default platforms that will be used for any folder
#    that is not explicitly listed below.
DEFAULT_PLATFORMS = ["linux/amd64","linux/arm64","linux/s390x","linux/ppc64le"]

# 2. Define a dictionary to customize platforms for specific folders.
#    The keys should be the folder names, and the values should be
#    a list of platforms for that folder.
CUSTOM_PLATFORMS = {
    "test-runner": ["linux/amd64"],
    "kubectl": ["linux/amd64"],
    "ko": ["linux/amd64"],
    "ko-gcloud": ["linux/amd64"],
    "coverage": ["linux/amd64"],
}

def generate_github_matrix(path):
    """
    Generates a GitHub Actions workflow matrix JSON from subdirectories in a given path,
    with customizable platforms for specific folders. The platforms are output as
    a single, comma-separated string.

    Args:
        path (str): The path to the directory containing the subdirectories.

    Returns:
        str: A JSON string representing the GitHub Actions matrix.
    """
    matrix_items = []

    if not os.path.isdir(path):
        print(f"Error: The provided path '{path}' is not a valid directory.")
        return json.dumps({"include": []})

    for dir_name in os.listdir(path):
        full_path = os.path.join(path, dir_name)
        
        # We only want to process subdirectories.
        if os.path.isdir(full_path):
            
            # Check if the folder name is in our customization dictionary.
            if dir_name in CUSTOM_PLATFORMS:
                # Use the custom platforms for this specific folder.
                platforms_for_folder = CUSTOM_PLATFORMS[dir_name]
            else:
                # Use the default platforms for all other folders.
                platforms_for_folder = DEFAULT_PLATFORMS
            
            # Concatenate the list of platforms into a single, comma-separated string.
            platforms_string = ",".join(platforms_for_folder)
            
            item = {
                "name": dir_name,
                "platforms": platforms_string
            }
            
            matrix_items.append(item)

    matrix_output = {
        "include": matrix_items
    }

    return json.dumps(matrix_output, indent=2)

if __name__ == "__main__":
    # Check if a command-line argument for the directory path was provided.
    if len(sys.argv) > 1:
        # The first argument (sys.argv[1]) is our target directory.
        target_directory = sys.argv[1]
    else:
        # If no argument is provided, use the current directory as the default.
        target_directory = "."
    
    matrix_json = generate_github_matrix(target_directory)
    print(matrix_json)
