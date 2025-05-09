package main

import (
	"log/slog"
	"strconv"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	camera  rl.Camera3D
	numbers []rl.Texture2D
)

type BodyDrawOptions struct {
	ShowEdges    bool
	ShowVertexes bool
}

func main() {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(960, 540, "Body Deformation")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.SetWindowState(rl.FlagWindowResizable)

	camera = rl.NewCamera3D(
		rl.NewVector3(10, 10, 10),
		rl.NewVector3(0, 0, 0),
		rl.NewVector3(0, 1, 0),
		45,
		rl.CameraPerspective,
	)
	cameraOrbiting := true

	bodySize := [3]*InputValue[float64]{
		NewInputValue(4.0),
		NewInputValue(5.0),
		NewInputValue(3.0),
	}

	bodySplit := [3]*InputValue[int]{
		NewInputValue(4),
		NewInputValue(8),
		NewInputValue(3),
	}

	yungaModule := NewInputValue(4.0)
	poissonRatio := NewInputValue(0.3)
	pressure := NewInputValue(2.0)

	fem := &FEM{}
	body, bodyIndexes := fem.BuildElements(InputsToSlice3(bodySize), InputsToSlice3(bodySplit))
	var deformedBody [][3]float64

	const inputTextSize = 20
	var (
		inputWidth  float32 = 64.0
		inputHeight float32 = 24.0
		padding     float32 = 8.0
	)

	a0 := rl.NewVector3(0, 0, 0)
	aX := rl.NewVector3(1, 0, 0)
	aY := rl.NewVector3(0, 1, 0)
	aZ := rl.NewVector3(0, 0, 1)

	for i := range 1 << 11 {
		textImage := rl.ImageTextEx(rl.GetFontDefault(), strconv.Itoa(i), 32, 4, rl.White)
		numbers = append(numbers, rl.LoadTextureFromImage(textImage))
		rl.UnloadImage(textImage)
	}

	showNumbers := false
	showOriginal := true
	opt := BodyDrawOptions{
		ShowEdges:    true,
		ShowVertexes: true,
	}

	for !rl.WindowShouldClose() {
		topLeftUiRect := rl.NewRectangle(
			0, 0,
			padding+inputWidth*3.5+padding*2.5+padding,
			padding+inputHeight*2+padding+padding,
		)
		bottomLeftUiRect := rl.NewRectangle(
			0, float32(rl.GetScreenHeight())-(padding+inputHeight*3+padding*2+padding),
			padding+inputWidth*2+padding+padding,
			padding+inputHeight*3+padding*2+padding,
		)

		if rl.IsKeyPressed(rl.KeySpace) {
			cameraOrbiting = !cameraOrbiting
		}

		if rl.IsMouseButtonDown(rl.MouseButtonLeft) && (!rl.CheckCollisionPointRec(rl.GetMousePosition(), topLeftUiRect) &&
			!rl.CheckCollisionPointRec(rl.GetMousePosition(), bottomLeftUiRect)) {
			md := rl.GetMouseDelta()
			rl.CameraYaw(&camera, -md.X*0.003, 1)
			rl.CameraPitch(&camera, -md.Y*0.003, 1, 1, 0)
		} else if cameraOrbiting {
			var rotation = rl.MatrixRotate(rl.GetCameraUp(&camera), 0.5*rl.GetFrameTime())
			var view = rl.Vector3Subtract(camera.Position, camera.Target)
			view = rl.Vector3Transform(view, rotation)
			camera.Position = rl.Vector3Add(camera.Target, view)
		}

		rl.CameraMoveToTarget(&camera, -rl.GetMouseWheelMove())
		if rl.IsKeyPressed(rl.KeyKpSubtract) {
			rl.CameraMoveToTarget(&camera, 2.0)
		}
		if rl.IsKeyPressed(rl.KeyKpAdd) {
			rl.CameraMoveToTarget(&camera, -2.0)
		}

		const scaleFactor = 0.95
		if rl.IsKeyPressed(rl.KeyMinus) {
			inputWidth *= scaleFactor
			inputHeight *= scaleFactor
			padding *= scaleFactor
		}
		if rl.IsKeyPressed(rl.KeyEqual) {
			inputWidth /= scaleFactor
			inputHeight /= scaleFactor
			padding /= scaleFactor
		}

		if rl.IsKeyPressed(rl.KeyN) {
			showNumbers = !showNumbers
		}
		if rl.IsKeyPressed(rl.KeyO) {
			showOriginal = !showOriginal
		}
		if rl.IsKeyPressed(rl.KeyE) {
			opt.ShowEdges = !opt.ShowEdges
		}
		if rl.IsKeyPressed(rl.KeyV) {
			opt.ShowVertexes = !opt.ShowVertexes
		}

		rl.BeginDrawing()
		{
			rl.ClearBackground(rl.RayWhite)

			rl.BeginMode3D(camera)
			{
				rl.DrawGrid(32, 1)

				origin := rl.Vector3Scale(InputsToVec3(bodySize), 0.5)
				origin.Y, origin.Z = origin.Z, origin.Y
				origin.Y = 0

				if showOriginal {
					drawBody(body, bodyIndexes, origin, rl.Gray, rl.Blue, showNumbers, opt)
				}
				if deformedBody != nil {
					drawBody(deformedBody, bodyIndexes, origin, rl.Red, rl.Green, false, opt)
				}

				const thickness = 0.02
				rl.DrawCylinderEx(a0, aX, thickness, thickness, 8, rl.Red)
				rl.DrawCylinderEx(a0, aY, thickness, thickness, 8, rl.Green)
				rl.DrawCylinderEx(a0, aZ, thickness, thickness, 8, rl.Blue)
			}
			rl.EndMode3D()

			rl.DrawRectangleRec(topLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(topLeftUiRect, 1, rl.Gray)

			rl.DrawRectangleRec(bottomLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(bottomLeftUiRect, 1, rl.Gray)

			bodyUpdated := false

			// Sizes
			gui.Label(rl.NewRectangle(padding, padding, inputWidth/2, inputHeight), "Size")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding, inputWidth, inputHeight),
					&bodySize[i].Text, inputTextSize, bodySize[i].Edit,
				) {
					bodySize[i].ToggleEdit()
					v, err := strconv.ParseFloat(bodySize[i].Text, 32)
					if err != nil {
						slog.Error("Invalid size value", "err", err)
					} else {
						v = max(min(v, 1000), 0.01)
						bodySize[i].Value = v
						bodyUpdated = true
					}
					bodySize[i].UpdateText()
				}
			}

			// Splits
			gui.Label(rl.NewRectangle(padding, padding+padding+inputHeight, inputWidth/2, inputHeight), "Split")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding+inputHeight+padding, inputWidth, inputHeight),
					&bodySplit[i].Text, inputTextSize, bodySplit[i].Edit,
				) {
					bodySplit[i].ToggleEdit()
					v, err := strconv.ParseInt(bodySplit[i].Text, 10, 32)
					if err != nil {
						slog.Error("Invalid split value", "err", err, "value", v)
					} else {
						v = max(min(v, 100), 1)
						bodySplit[i].Value = int(v)
						bodyUpdated = true
					}
					bodySplit[i].UpdateText()
				}
			}

			// Young's modulus
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight), "Young's Modulus")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight),
				&yungaModule.Text, inputTextSize, yungaModule.Edit,
			) {
				yungaModule.ToggleEdit()
				v, err := strconv.ParseFloat(yungaModule.Text, 64)
				if err != nil {
					slog.Error("Invalid Young's modulus value", "err", err)
				} else {
					yungaModule.Value = max(min(v, 100000.0), 0.01)
				}
				yungaModule.UpdateText()
			}

			// Poisson's ratio
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight), "Poisson's ratio")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight),
				&poissonRatio.Text, inputTextSize, poissonRatio.Edit,
			) {
				poissonRatio.ToggleEdit()
				v, err := strconv.ParseFloat(poissonRatio.Text, 64)
				if err != nil {
					slog.Error("Invalid Poisson's ratio value", "err", err)
				} else {
					poissonRatio.Value = max(min(v, 0.49), 0.0)
				}
				poissonRatio.UpdateText()
			}

			// Pressure
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight), "Pressure")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight),
				&pressure.Text, inputTextSize, pressure.Edit,
			) {
				pressure.ToggleEdit()
				v, err := strconv.ParseFloat(pressure.Text, 64)
				if err != nil {
					slog.Error("Invalid pressure value", "err", err)
				} else {
					pressure.Value = max(min(v, 10000.0), 0.01)
				}
				pressure.UpdateText()
			}

			if bodyUpdated {
				body, bodyIndexes = fem.BuildElements(InputsToSlice3(bodySize), InputsToSlice3(bodySplit))
				deformedBody = nil
			}

			// Run
			if gui.Button(
				rl.NewRectangle(float32(rl.GetScreenWidth())-padding-inputWidth, float32(rl.GetScreenHeight())-padding-inputHeight, inputWidth, inputHeight),
				"Run",
			) {
				slog.Info("Running...",
					"bodySize", InputsToVec3(bodySize),
					"bodySplits", InputsToVec3(bodySplit),
					"yungaModule", yungaModule, "poissonRatio", poissonRatio, "pressure", pressure,
				)
				fem.ChoseConditions(InputsToSlice3(bodySplit))
				deformedBody = fem.ApplyForce(yungaModule.Value, poissonRatio.Value, pressure.Value)
			}
		}
		rl.EndDrawing()
	}
}

func drawBody(
	body [][3]float64, bodyIndexes map[[3]int]int, origin rl.Vector3,
	edgesColor, verticesColor rl.Color, showNumbers bool, opt BodyDrawOptions,
) {
	for key, idx := range bodyIndexes {
		p1 := body[idx]
		pv1 := rl.Vector3Subtract(rl.NewVector3(float32(p1[0]), float32(p1[2]), float32(p1[1])), origin)

		const vertexSize = 0.07
		if opt.ShowVertexes {
			rl.DrawCube(pv1, vertexSize, vertexSize, vertexSize, verticesColor)
		}

		if opt.ShowEdges {
			i, j, k := key[0], key[1], key[2]
			for _, delta := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
				ni, nj, nk := i+delta[0], j+delta[1], k+delta[2]
				neighborKey := [3]int{ni, nj, nk}
				if nIdx, ok := bodyIndexes[neighborKey]; ok {
					p2 := body[nIdx]
					pv2 := rl.Vector3Subtract(rl.NewVector3(float32(p2[0]), float32(p2[2]), float32(p2[1])), origin)
					rl.DrawLine3D(pv1, pv2, edgesColor)
				}
			}
		}

		if showNumbers {
			rl.DrawBillboard(camera, numbers[idx+1], rl.Vector3Add(pv1, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
		}
	}
}
