
from PIL import Image, ImageDraw, ImageFont

def add_text_to_texture(image_path, output_image_path, text):
    # Open the image (texture)
    img = Image.open(image_path)

    # Ensure image is in RGB mode (if it's not already)
    if img.mode != 'RGB':
        img = img.convert('RGB')
        
    # Initialize the drawing context
    draw = ImageDraw.Draw(img)
    
    # Define font and size (optional)
    font = ImageFont.load_default()

    # Add text to the image (adjust position as needed)
    draw.text((500, 500), text, font=font, fill="green")

    # Save the updated image
    img.save(output_image_path)
    print(f"Updated texture saved to {output_image_path}")

# Example usage
add_text_to_texture('../public/CASETTE_MODEL/Textures/(1)lowPolyExploded1_Casette_4K_BaseColor.1001.png', './new-texture.png', 'New Text')

